## ADDED Requirements

### Requirement: Sensitive word management
The system SHALL allow administrators to manage sensitive words through CRUD operations.

#### Scenario: List sensitive words with pagination
- **WHEN** admin sends GET /api/v1/sensitive/words with page and page_size params
- **THEN** system returns paginated list of sensitive words with total count

#### Scenario: Create a sensitive word
- **WHEN** admin sends POST /api/v1/sensitive/words with word, strategy, and optional replacement
- **THEN** system creates the word and refreshes the DFA engine

#### Scenario: Update a sensitive word
- **WHEN** admin sends PUT /api/v1/sensitive/words/:word_id with fields to update
- **THEN** system updates the word record

#### Scenario: Delete a sensitive word
- **WHEN** admin sends DELETE /api/v1/sensitive/words/:word_id
- **THEN** system deletes the word and refreshes the DFA engine

#### Scenario: Batch import sensitive words
- **WHEN** admin sends POST /api/v1/sensitive/words/batch with array of words
- **THEN** system imports all words and refreshes the DFA engine

### Requirement: Sensitive word detection via DFA
The system SHALL detect sensitive words in text using a DFA trie with O(n) time complexity.

#### Scenario: No sensitive word found returns passed
- **WHEN** user sends text with no sensitive words to POST /api/v1/sensitive/check/text
- **THEN** system returns passed=true

#### Scenario: Sensitive word hit returns matched words
- **WHEN** user sends text containing a configured sensitive word
- **THEN** system returns the matched words and their strategies

#### Scenario: Case-insensitive matching
- **WHEN** user sends text with uppercase variations of a sensitive word
- **THEN** system still detects the match

#### Scenario: Replace strategy modifies text
- **WHEN** user sends text containing a word with strategy=replace
- **THEN** system returns cleaned text with replacement characters

#### Scenario: Block strategy rejects message
- **WHEN** user sends text containing a word with strategy=block
- **THEN** system returns blocked=true and passed=false

#### Scenario: Log strategy logs without modification
- **WHEN** user sends text containing a word with strategy=log
- **THEN** system returns the word hit but passed=true

### Requirement: DFA engine hot-reload
The system SHALL support rebuilding the DFA trie after word list changes without service restart.

#### Scenario: DFA refresh after word CRUD
- **WHEN** a word is created, updated, or deleted
- **THEN** system asynchronously rebuilds the DFA engine within 10 seconds
