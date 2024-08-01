# Lambda Testing Guidelines

This document outlines our approach to testing the Lambda functions and associated components in our project.

## Running Tests

To run all tests in the project:

```bash
go test -v ./functions/...
```

To run tests for a specific package:

```bash
go test ./functions/lambda/helpers
```

To run tests with coverage:

```bash
go test -v -cover ./functions/lambda/...
```

## Types of Tests

### 1. Unit Tests

We use unit tests to verify individual functions and methods. Examples include:

#### Helpers Package (`helpers_test.go`):

```go
func TestFormatDate(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"Valid date", "2099-05-01T12:00:00Z", "May 1, 2099 (Fri)"},
        {"Invalid date", "invalid-date", "Invalid date"},
        {"Empty string", "", "Invalid date"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FormatDate(tt.input)
            if result != tt.expected {
                t.Errorf("FormatDate(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}
```

#### Indexing Package (`z_order_index_test.go`):

```go
func TestCalculateZOrderIndex(t *testing.T) {
    tests := []struct {
        name      string
        timestamp time.Time
        lat       float32
        lon       float32
        indexType string
        want      []byte
    }{
        // Test cases here
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CalculateZOrderIndex(tt.timestamp, tt.lat, tt.lon, tt.indexType)
            if err != nil {
                t.Errorf("CalculateZOrderIndex() error = %v", err)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("CalculateZOrderIndex() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 2. Integration Tests

Integration tests verify the interaction between different components. Examples include:

#### Handlers Package (`data_handlers_test.go`):

```go
func TestCreateEvent(t *testing.T) {
    mockDB := &mockDynamoDBClient{
        putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
            // Mock implementation
        },
    }

    tests := []struct {
        name           string
        requestBody    string
        expectedStatus int
    }{
        // Test cases here
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := events.APIGatewayV2HTTPRequest{Body: tt.requestBody}
            resp, err := CreateEvent(context.Background(), req, mockDB)
            // Assert response and error
        })
    }
}
```

## Test Coverage

We aim for a minimum test coverage of 50%. To check coverage:

```bash
go test -coverprofile=coverage.out ./functions/lambda/...
go tool cover -func=coverage.out
```

To view coverage in HTML format:

```bash
go tool cover -html=coverage.out -o coverage.html
```

## Continuous Integration

Our GitHub Actions workflow (`go.yml`) automatically runs tests and checks coverage for each push and pull request. The workflow fails if coverage falls below 50%.

Key parts of the workflow:

```yaml
- name: Run tests and generate coverage
  run: |
    go test -v -coverprofile=coverage.out ./functions/lambda/...
    go tool cover -func=coverage.out > coverage.txt

- name: Check coverage
  run: |
    COVERAGE=$(grep -Po '(?<=total:\s{6})\d+\.\d+' coverage.txt)
    echo "Total coverage: $COVERAGE%"
    if (( $(echo "$COVERAGE < 50" | bc -l) )); then
      echo "Coverage is below 50%"
      exit 1
    fi
```

## Best Practices

1. Use table-driven tests for functions with multiple input scenarios.
2. Mock external dependencies (e.g., DynamoDB) in tests.
3. Test both happy paths and error scenarios.
4. Keep test files in the same package as the code they're testing.
5. Use meaningful test names that describe the scenario being tested.
6. Regularly review and update tests as the codebase evolves.

## Troubleshooting

If tests are failing in CI but passing locally:
1. Ensure all dependencies are up to date.
2. Check for environment-specific issues (e.g., file paths, environment variables).
3. Review the CI logs for any system-level differences.
