# bm25-search Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Document indexing

The system SHALL index memory entry content for BM25 full-text search.

#### Scenario: Index new document
- **WHEN** a memory entry is stored with text content
- **THEN** the system tokenizes and indexes the content for BM25 search

#### Scenario: Update document index
- **WHEN** a memory entry's content is updated
- **THEN** the system removes the old index and creates a new index for the updated content

### Requirement: Text tokenization

The system SHALL tokenize text content into terms for indexing and searching.

#### Scenario: Tokenize English text
- **WHEN** indexing English text "Hello World"
- **THEN** the system produces tokens ["hello", "world"] (lowercase)

#### Scenario: Tokenize with punctuation removal
- **WHEN** indexing text with punctuation "Hello, World!"
- **THEN** the system produces tokens ["hello", "world"] without punctuation

#### Scenario: Tokenize Chinese text
- **WHEN** indexing Chinese text "浣犲ソ涓栫晫"
- **THEN** the system produces appropriate Chinese tokens

### Requirement: IDF calculation

The system SHALL calculate Inverse Document Frequency (IDF) for all terms in the corpus.

#### Scenario: Calculate IDF for common term
- **WHEN** a term appears in many documents
- **THEN** the system assigns a low IDF score

#### Scenario: Calculate IDF for rare term
- **WHEN** a term appears in few documents
- **THEN** the system assigns a high IDF score

### Requirement: BM25 scoring

The system SHALL calculate BM25 scores using the formula: BM25(q,d) = 危 IDF(qi) * (f(qi,d) * (k1+1)) / (f(qi,d) + k1 * (1-b+b*|d|/avgdl))

#### Scenario: Score document with query match
- **WHEN** a document contains all query terms
- **THEN** the system calculates a positive BM25 score

#### Scenario: Score document without query match
- **WHEN** a document contains no query terms
- **THEN** the system returns a BM25 score of 0

### Requirement: BM25 parameters

The system SHALL support configurable BM25 parameters k1 and b.

#### Scenario: Use default parameters
- **WHEN** BM25 is initialized without custom parameters
- **THEN** the system uses k1=1.5 and b=0.75 (standard values)

#### Scenario: Use custom parameters
- **WHEN** BM25 is initialized with k1=2.0 and b=0.5
- **THEN** the system uses the custom parameters for scoring

### Requirement: Top-K search results

The system SHALL return top-K documents ranked by BM25 score.

#### Scenario: Search with top-10 limit
- **WHEN** searching with query "machine learning" and topK=10
- **THEN** the system returns up to 10 documents ranked by BM25 score in descending order

#### Scenario: Search with fewer results than K
- **WHEN** searching with topK=10 but only 5 documents match
- **THEN** the system returns all 5 matching documents

### Requirement: Session-based search filtering

The system SHALL filter BM25 search results by SessionID.

#### Scenario: Search within session
- **WHEN** searching for session "A"
- **THEN** the system returns only documents belonging to session "A"

#### Scenario: Search across all sessions
- **WHEN** searching with empty session filter
- **THEN** the system returns documents from all sessions

### Requirement: Average document length tracking

The system SHALL track and update average document length for BM25 calculation.

#### Scenario: Update average on document addition
- **WHEN** a new document is indexed
- **THEN** the system recalculates the average document length

#### Scenario: Update average on document removal
- **WHEN** a document is removed from the index
- **THEN** the system recalculates the average document length

### Requirement: Stop words handling

The system SHALL support optional stop words filtering during tokenization.

#### Scenario: Filter common stop words
- **WHEN** stop words filtering is enabled and text contains "the", "a", "is"
- **THEN** the system excludes these stop words from indexing

#### Scenario: No stop words filtering
- **WHEN** stop words filtering is disabled
- **THEN** the system indexes all terms including common words

### Requirement: BM25 search performance

The system SHALL perform BM25 search with sub-10ms latency for corpora up to 100K documents.

#### Scenario: Search in small corpus
- **WHEN** searching in a corpus with 1K documents
- **THEN** the search completes in less than 5ms

#### Scenario: Search in large corpus
- **WHEN** searching in a corpus with 100K documents
- **THEN** the search completes in less than 10ms

