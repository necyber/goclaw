# TLS/mTLS Certificate Setup

This guide explains how to set up TLS and mutual TLS (mTLS) for secure gRPC communication in Goclaw.

## Overview

- **TLS**: Server authentication - clients verify the server's identity
- **mTLS**: Mutual authentication - both client and server verify each other's identity

## Prerequisites

- OpenSSL installed
- Basic understanding of PKI (Public Key Infrastructure)

## Quick Start - Self-Signed Certificates

For development and testing, you can generate self-signed certificates:

### 1. Create Certificate Directory

```bash
mkdir -p certs
cd certs
```

### 2. Generate CA (Certificate Authority)

```bash
# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate CA certificate
openssl req -new -x509 -key ca.key -sha256 -subj "/C=US/ST=CA/O=Goclaw/CN=Goclaw CA" -days 3650 -out ca.crt
```

### 3. Generate Server Certificate

```bash
# Generate server private key
openssl genrsa -out server.key 4096

# Generate server CSR (Certificate Signing Request)
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=CA/O=Goclaw/CN=localhost"

# Create server certificate extensions file
cat > server.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Sign server certificate with CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days 365 -sha256 -extfile server.ext

# Clean up
rm server.csr server.ext
```

### 4. Generate Client Certificate (for mTLS)

```bash
# Generate client private key
openssl genrsa -out client.key 4096

# Generate client CSR
openssl req -new -key client.key -out client.csr -subj "/C=US/ST=CA/O=Goclaw/CN=client"

# Sign client certificate with CA
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out client.crt -days 365 -sha256

# Clean up
rm client.csr
```

### 5. Verify Certificates

```bash
# Verify server certificate
openssl verify -CAfile ca.crt server.crt

# Verify client certificate
openssl verify -CAfile ca.crt client.crt

# View certificate details
openssl x509 -in server.crt -text -noout
```

## Configuration

### TLS Configuration (Server Authentication Only)

Update `config/config.yaml`:

```yaml
server:
  grpc:
    enabled: true
    port: 9090

    tls:
      enabled: true
      cert_file: "./certs/server.crt"
      key_file: "./certs/server.key"
      ca_file: "./certs/ca.crt"
      client_auth: false  # TLS only, no client verification
```

### mTLS Configuration (Mutual Authentication)

Update `config/config.yaml`:

```yaml
server:
  grpc:
    enabled: true
    port: 9090

    tls:
      enabled: true
      cert_file: "./certs/server.crt"
      key_file: "./certs/server.key"
      ca_file: "./certs/ca.crt"
      client_auth: true  # Enable mTLS
```

## Testing

### Test with grpcurl

#### TLS (Server Authentication)

```bash
grpcurl -cacert ./certs/ca.crt \
  localhost:9090 list
```

#### mTLS (Mutual Authentication)

```bash
grpcurl -cacert ./certs/ca.crt \
  -cert ./certs/client.crt \
  -key ./certs/client.key \
  localhost:9090 list
```

### Test with Go Client

#### TLS

```go
import "github.com/goclaw/goclaw/pkg/grpc/client"

c, err := client.NewClient("localhost:9090",
    client.WithTLS("./certs/ca.crt", "", ""),
)
```

#### mTLS

```go
c, err := client.NewClient("localhost:9090",
    client.WithTLS("./certs/ca.crt", "./certs/client.crt", "./certs/client.key"),
)
```

## Production Setup

For production environments, use certificates from a trusted Certificate Authority (CA) like Let's Encrypt, DigiCert, or your organization's internal CA.

### Using Let's Encrypt

```bash
# Install certbot
sudo apt-get install certbot

# Generate certificate (HTTP-01 challenge)
sudo certbot certonly --standalone -d your-domain.com

# Certificates will be in:
# /etc/letsencrypt/live/your-domain.com/fullchain.pem  # Server cert
# /etc/letsencrypt/live/your-domain.com/privkey.pem    # Server key
# /etc/letsencrypt/live/your-domain.com/chain.pem      # CA cert
```

Update configuration:

```yaml
server:
  grpc:
    tls:
      enabled: true
      cert_file: "/etc/letsencrypt/live/your-domain.com/fullchain.pem"
      key_file: "/etc/letsencrypt/live/your-domain.com/privkey.pem"
      ca_file: "/etc/letsencrypt/live/your-domain.com/chain.pem"
```

### Certificate Rotation

For automatic certificate rotation:

1. **Use cert-manager** (Kubernetes):
   ```yaml
   apiVersion: cert-manager.io/v1
   kind: Certificate
   metadata:
     name: goclaw-tls
   spec:
     secretName: goclaw-tls-secret
     issuerRef:
       name: letsencrypt-prod
       kind: ClusterIssuer
     dnsNames:
       - goclaw.example.com
   ```

2. **Use systemd timer** (Linux):
   ```bash
   # Create renewal script
   cat > /usr/local/bin/renew-certs.sh << 'EOF'
   #!/bin/bash
   certbot renew --quiet
   systemctl reload goclaw
   EOF

   chmod +x /usr/local/bin/renew-certs.sh

   # Create systemd timer
   cat > /etc/systemd/system/cert-renewal.timer << 'EOF'
   [Unit]
   Description=Certificate Renewal Timer

   [Timer]
   OnCalendar=daily
   Persistent=true

   [Install]
   WantedBy=timers.target
   EOF

   systemctl enable cert-renewal.timer
   systemctl start cert-renewal.timer
   ```

## Security Best Practices

### Certificate Management

1. **Protect Private Keys**
   ```bash
   chmod 600 certs/*.key
   chown goclaw:goclaw certs/*.key
   ```

2. **Use Strong Key Sizes**
   - Minimum 2048 bits for RSA
   - Recommended 4096 bits for long-term use
   - Consider ECDSA for better performance

3. **Set Appropriate Validity Periods**
   - Development: 1 year
   - Production: 90 days (with auto-renewal)

4. **Regular Rotation**
   - Rotate certificates before expiration
   - Rotate immediately if compromised

### Server Configuration

1. **Disable Weak Cipher Suites**
   ```go
   // In server configuration
   tlsConfig := &tls.Config{
       MinVersion: tls.VersionTLS13,
       CipherSuites: []uint16{
           tls.TLS_AES_128_GCM_SHA256,
           tls.TLS_AES_256_GCM_SHA384,
           tls.TLS_CHACHA20_POLY1305_SHA256,
       },
   }
   ```

2. **Enable OCSP Stapling**
   ```go
   tlsConfig.OCSPStapling = true
   ```

3. **Use Certificate Pinning** (for high-security environments)

### Client Configuration

1. **Verify Server Hostname**
   ```go
   tlsConfig := &tls.Config{
       ServerName: "goclaw.example.com",
   }
   ```

2. **Don't Skip Verification**
   ```go
   // NEVER do this in production:
   tlsConfig.InsecureSkipVerify = true  // ❌ INSECURE
   ```

## Troubleshooting

### Common Issues

#### Certificate Verification Failed

```
Error: x509: certificate signed by unknown authority
```

**Solution**: Ensure CA certificate is correctly specified and trusted.

```bash
# Verify certificate chain
openssl verify -CAfile ca.crt server.crt
```

#### Hostname Mismatch

```
Error: x509: certificate is valid for localhost, not 192.168.1.100
```

**Solution**: Add IP address to certificate SAN (Subject Alternative Name):

```bash
# In server.ext
[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = 192.168.1.100  # Add your IP
```

#### Certificate Expired

```
Error: x509: certificate has expired
```

**Solution**: Generate new certificates or renew existing ones.

```bash
# Check expiration
openssl x509 -in server.crt -noout -dates
```

#### Permission Denied

```
Error: open ./certs/server.key: permission denied
```

**Solution**: Fix file permissions:

```bash
chmod 600 certs/*.key
chown goclaw:goclaw certs/*
```

### Debug TLS Connection

```bash
# Test TLS handshake
openssl s_client -connect localhost:9090 -CAfile certs/ca.crt

# With client certificate (mTLS)
openssl s_client -connect localhost:9090 \
  -CAfile certs/ca.crt \
  -cert certs/client.crt \
  -key certs/client.key
```

### Enable Debug Logging

```bash
# Server side
GRPC_GO_LOG_VERBOSITY_LEVEL=99 GRPC_GO_LOG_SEVERITY_LEVEL=info ./goclaw

# Client side
GRPC_GO_LOG_VERBOSITY_LEVEL=99 GRPC_GO_LOG_SEVERITY_LEVEL=info grpcurl ...
```

## Advanced Topics

### Using Hardware Security Modules (HSM)

For high-security environments, store private keys in HSM:

```go
import "crypto/x509"
import "github.com/ThalesIgnite/crypto11"

// Initialize PKCS#11 module
ctx, err := crypto11.Configure(&crypto11.Config{
    Path:       "/usr/lib/softhsm/libsofthsm2.so",
    TokenLabel: "goclaw",
    Pin:        "1234",
})

// Use HSM-backed private key
signer, err := ctx.FindKeyPair(nil, []byte("server-key"))
```

### Certificate Transparency

Enable CT logging for public certificates:

```bash
# Check CT logs
curl -s "https://crt.sh/?q=goclaw.example.com&output=json" | jq
```

### SPIFFE/SPIRE Integration

For service mesh environments:

```yaml
# SPIRE agent configuration
agent {
  data_dir = "/opt/spire/data/agent"
  server_address = "spire-server"
  server_port = "8081"
  trust_domain = "goclaw.example.com"
}
```

## References

- [gRPC Authentication Guide](https://grpc.io/docs/guides/auth/)
- [OpenSSL Documentation](https://www.openssl.org/docs/)
- [Let's Encrypt](https://letsencrypt.org/)
- [SPIFFE/SPIRE](https://spiffe.io/)

## See Also

- [gRPC Examples](./grpc-examples.md)
- [Client SDK Examples](./client-sdk-examples.md)
- [Security Best Practices](../security.md)
