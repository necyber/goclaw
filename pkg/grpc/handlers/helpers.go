package handlers

import (
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// convertToProtoStatus converts string status to proto WorkflowStatus
func convertToProtoStatus(status string) pb.WorkflowStatus {
	switch status {
	case "PENDING":
		return pb.WorkflowStatus_WORKFLOW_STATUS_PENDING
	case "RUNNING":
		return pb.WorkflowStatus_WORKFLOW_STATUS_RUNNING
	case "COMPLETED":
		return pb.WorkflowStatus_WORKFLOW_STATUS_COMPLETED
	case "FAILED":
		return pb.WorkflowStatus_WORKFLOW_STATUS_FAILED
	case "CANCELLED":
		return pb.WorkflowStatus_WORKFLOW_STATUS_CANCELLED
	default:
		return pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED
	}
}

// convertToProtoTaskStatus converts string status to proto TaskStatus
func convertToProtoTaskStatus(status string) pb.TaskStatus {
	switch status {
	case "PENDING":
		return pb.TaskStatus_TASK_STATUS_PENDING
	case "RUNNING":
		return pb.TaskStatus_TASK_STATUS_RUNNING
	case "COMPLETED":
		return pb.TaskStatus_TASK_STATUS_COMPLETED
	case "FAILED":
		return pb.TaskStatus_TASK_STATUS_FAILED
	case "CANCELLED":
		return pb.TaskStatus_TASK_STATUS_CANCELLED
	default:
		return pb.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// timestampFromUnix converts unix timestamp to proto timestamp
func timestampFromUnix(unix int64) *timestamppb.Timestamp {
	if unix == 0 {
		return nil
	}
	return timestamppb.New(time.Unix(unix, 0))
}
