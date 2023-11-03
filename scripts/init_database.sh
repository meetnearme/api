#!/bin/bash

export AWS_SECRET_ACCESS_KEY=""
export AWS_ACCESS_KEY_ID=""
export AWS_REGION=us-east-1

aws dynamodb create-table --cli-input-json file://internal/database/db_seeds/create_event_table.json --endpoint-url http://localhost:8000
aws dynamodb batch-write-item --cli-input-json file://internal/database/db_seeds/seed_user_records.json --endpoint-url http://localhost:8000

