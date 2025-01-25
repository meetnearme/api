## Users

1. Create User

```bash

# example of all fields
curl -X POST https://devnear.me/api/users \
 -H "Content-Type: application/json" \
 -d '{
     "name": "a Doe",
     "email": "a.doe@example.com",
     "role": "standard_user",
     "address": "123 Main St, New York 10001 USA",
     "category_preferences": "[\"sports\", \"music\"]",
     "phone": "+123456789",
     "profilePictureUrl": "https://example.com/profile.jpg"
 }'

 # only required fields
 curl -X POST  https://devnear.me/api/users \
 -H "Content-Type: application/json" \
 -d '{
     "name": "John Doe",
     "email": "john.doe@example.com",
     "role": "standard_user"
 }'

 
```

2. Get User by ID
```bash
curl -X GET https://devnear.me/api/users/<:user_id>

```

3. Get Users
```bash
curl -X GET https://devnear.me/api/users

```

3. Update user
```bash
curl -X PUT https://devnear.me/api/users/<:user_id> \
-H "Content-Type: application/json" \
-d '{
    "name": "New name",
    "email": "new.name@example.com",
    "address_street": "51 main street",
    "address_city": "New Haven",
    "category_preferences": "['sports', 'music', 'clubs']",
    "address_zip_code": "51515",
    "address_country": "USA",
    "phone": "+1234567890",
    "profile_picture_url": "http://example.com/profile.jpg",
    "role": "organization_user"
}'

```

4. Delete User
```bash
curl -X DELETE https://devnear.me/api/users/<:user_id> 
```

## Purchasables

1. Create Purchasable
```bash
curl -X POST https://byddmq7zrb.execute-api.us-east-1.amazonaws.com/api/purchasables/123e4567-e89b-12d3-a456-426614174000 \
-H "Content-Type: application/json" \
-d '{
  "event_id": "123e4567-e89b-12d3-a456-426614174000",
  "registration_fields": ["field1", "field2"],
  "purchasable_items": [
    {
      "name": "Sample Item",
      "item_type": "Type A",
      "cost": 100.0,
      "inventory": 50,
      "starting_quantity": 100,
      "currency": "USD",
      "charge_recurrence_interval": "monthly",
      "charge_recurrence_interval_count": 3,
      "charge_recurrence_end_date": "2025-12-31T00:00:00Z",
      "donation_ratio": 0.1,
      "created_at": "2024-09-01T12:00:00Z",
      "updated_at": "2024-09-01T12:00:00Z"
    }
  ],
  "created_at": "2024-09-01T12:00:00Z",
  "updated_at": "2024-09-01T12:00:00Z"
}'

```

2. Update Purchasable
```bash
curl -X PUT https://byddmq7zrb.execute-api.us-east-1.amazonaws.com/api/purchasables/123e4567-e89b-12d3-a456-426614174000 \
-H "Content-Type: application/json" \
-d '{
  "event_id": "123e4567-e89b-12d3-a456-426614174000",
  "registration_fields": ["field1", "field2"],
  "purchasable_items": [
    {
      "name": "Sample Item",
      "item_type": "Type A",
      "cost": 100.0,
      "inventory": 50,
      "starting_quantity": 100,
      "currency": "USD",
      "charge_recurrence_interval": "monthly",
      "charge_recurrence_interval_count": 3,
      "charge_recurrence_end_date": "2025-12-31T00:00:00Z",
      "donation_ratio": 0.1,
      "created_at": "2024-09-01T12:00:00Z",
      "updated_at": "2024-09-01T12:00:00Z"
    },
    {
      "name": "Updated Item",
      "item_type": "Type B",
      "cost": 150.0,
      "inventory": 30,
      "starting_quantity": 80,
      "currency": "USD",
      "charge_recurrence_interval": "yearly",
      "charge_recurrence_interval_count": 1,
      "charge_recurrence_end_date": "2026-12-31T00:00:00Z",
      "donation_ratio": 0.2,
      "updated_at": "2024-09-15T14:30:00Z"
    }
  ],
  "updated_at": "2024-09-15T14:30:00Z"
}'

```
3. Get Purchasables by EventID
```bash
curl -X GET https://byddmq7zrb.execute-api.us-east-1.amazonaws.com/api/purchasables/123e4567-e89b-12d3-a456-426614174000 \
-H "Content-Type: application/json"

```
5. Delete Purchasable
```bash
curl -X DELETE https://byddmq7zrb.execute-api.us-east-1.amazonaws.com/api/purchasables/123e4567-e89b-12d3-a456-426614174000 \
-H "Content-Type: application/json"
```


## Event RSVPs
1. Create EventRsvp
```bash
curl -X POST https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/event-rsvps/6ce1be30-f700-475c-b84a-49af0c73f337/ea49a5f8-e27c-47b0-8237-6f6f380a048c \
  -H "Content-Type: application/json" \
  -d '{
    "event_source_id": "71b19c4a-4390-426c-bbe0-77f214a90cfc",
    "event_source_type": "internalRecurrence",
    "status": "Yes"
}'
```

2. GET EventRsvp By PK 
```bash
curl -X GET https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/event-rsvps/6ce1be30-f700-475c-b84a-49af0c73f337/ea49a5f8-e27c-47b0-8237-6f6f380a048c \
  -H "Content-Type: application/json"

```

3. GET EventRsvp By UserID
```bash
curl -X GET https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/event-rsvps/user/ea49a5f8-e27c-47b0-8237-6f6f380a048c \
  -H "Content-Type: application/json"

```
4. GET EventRsvp By EventID
```bash
curl -X GET https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/event-rsvps/event/6ce1be30-f700-475c-b84a-49af0c73f337 \
  -H "Content-Type: application/json"
```

5. Update EventRsvp
```bash
curl -X PUT https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/event-rsvps/6ce1be30-f700-475c-b84a-49af0c73f337/ea49a5f8-e27c-47b0-8237-6f6f380a048c \
  -H "Content-Type: application/json" \
  -d '{
    "status": "Maybe"
}'

```

6. Delete EventRsvp
```bash
curl -X DELETE https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/event-rsvps/6ce1be30-f700-475c-b84a-49af0c73f337/ea49a5f8-e27c-47b0-8237-6f6f380a048c \
  -H "Content-Type: application/json"

```

# DynamoDB Endpoints

## Registration Fields
1. Create Registration Fields
```bash
curl -X POST https://devnear.me/api/registration-fields/<:event_id> \
-H "Content-Type: application/json" \
-d '{
  "updated_by": "Jim Bobby",
  "fields": [
    {
      "name": "attendeeEmail",
      "type": "text",
      "required": true,
      "default": "",
      "placeholder": "email@example.com",
      "description": "We need your email to contact you if the event is cancelled"
    },
    {
      "name": "tshirtSize",
      "type": "select",
      "required": true,
      "default": "medium",
      "placeholder": "",
      "description": "We need your tshirt size to order your shirt",
      "options": ["medium", "large", "XL"]
    }
  ]
}'

```

2. Get Registration Fields
```bash
curl -X GET https://devnear.me/api/registration-fields/<:event_id> \
-H "Content-Type: application/json"
```

3. Update Registration Fields
```bash
curl -X PUT https://devnear.me/api/registration-fields/<:event_id> \
-H "Content-Type: application/json" \
-d '{
  "updated_by": "Bobby Thyme",
  "fields": [
    {
      "name": "attendeeEmail",
      "type": "text",
      "required": true,
      "default": "",
      "placeholder": "email@example.com",
      "description": "We need your updated email in case of any changes"
    },
    {
      "name": "tshirtSize",
      "type": "select",
      "required": true,
      "default": "large",
      "placeholder": "",
      "description": "We need your updated tshirt size for the event",
      "options": ["small", "medium", "large", "XL"]
    },
    {
      "name": "sessionPreference",
      "type": "select",
      "required": true,
      "default": "morning",
      "placeholder": "",
      "description": "Please choose your preferred session",
      "options": ["morning", "evening"]
    }
  ]
}'
```

4. Delete Registration Fields
```bash
curl -X DELETE https://devnear.me/api/registration-fields/<:event_id> \
-H "Content-Type: application/json"
```


## Registrations

#### Note eventId comes before userId in url params

1. Create Registration
```bash
curl -X POST https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/registrations/62352e94-b34d-4ee7-a9d1-f1c8e404dec0/99413f71-bb0e-43c9-bc3a-fafc64c5c799 \
     -H "Content-Type: application/json" \
     -d '{
           "responses": [
             {"attendeeEmail": "me@meetnear.ne"},
             {"tShirtSize": "XL"}
           ]
         }'
```
2. Get Registration by Primary Key
```bash
/api/registrations/{:event_id}/{:user_id}
curl -X GET https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/registrations/62352e94-b34d-4ee7-a9d1-f1c8e404dec0/c9413f71-bb0e-43c9-bc3a-fafc64c5c799 \
     -H "Content-Type: application/json"
```

3. Get Registration by EventId
```bash
curl -X GET https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/registrations/event/62352e94-b34d-4ee7-a9d1-f1c8e404dec0 \
     -H "Content-Type: application/json"
```

4. Get Registration by UserId
```bash
curl -X GET https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/registrations/user/c9413f71-bb0e-43c9-bc3a-fafc64c5c799 \
     -H "Content-Type: application/json"
```

5. Update registration (uses PK)
```bash
curl -X PUT https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/registrations/62352e94-b34d-4ee7-a9d1-f1c8e404dec0/c9413f71-bb0e-43c9-bc3a-fafc64c5c799 \
     -H "Content-Type: application/json" \
     -d '{
           "responses": [
             {"attendeeEmail": "newemail@meetnear.ne"},
             {"tShirtSize": "L"}
           ]
         }'
```

6. Delete Registration (uses PK)
```bash
curl -X DELETE https://v63ojpt121.execute-api.us-east-1.amazonaws.com/api/registrations/62352e94-b34d-4ee7-a9d1-f1c8e404dec0/c9413f71-bb0e-43c9-bc3a-fafc64c5c799 \
     -H "Content-Type: application/json"
```

## Competition Config

1. Create Competition Config
```bash
curl -X POST https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-config \
-H "Content-Type: application/json" \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-d '{
    "auxilaryOwners": ["000000000000000000", "111111111111111111"],
    "eventIds": ["62352e94-b34d-4ee7-a9d1-f1c8e404dec0", "99413f71-bb0e-43c9-bc3a-fafc64c5c799"],
    "name": "Summer Karaoke Contest Try Number 2",
    "moduleType": "KARAOKE",
    "scoringMethod": "VOTES",
    "competitors": ["333333333333333333", "444444444444444444", "555555555555555555"],
    "status": "DRAFT"
}'
```
```
Bearer cnwTeb4mR-qjbbrYuKcCuuxB8vIqhrzeFr9d35PHc_eA7ycDQ9mypVump9xst1Obff6sw4PNe9USOWjiuJF40Is3rYAnr2pY-nhTxOzG

```

2. Get Competition Config by Config Id
```bash
curl -X GET https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-config/1a7e4d2f-2450-41d1-af03-8b3aa714fe69 \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json"
```

3. Get Competition Config by User Id
```bash
curl -X GET https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-config/owner \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json"
```


4. Update Competition Config
```bash
curl -X PUT https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-config/134c1f79-a276-493b-87c2-c63f79c3dae1 \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json" \
-d '{
    "name": "Updated Karaoke Contest",
    "status": "ACTIVE",
    "auxilaryOwners": ["ea49a5f8-e27c-47b0-8237-6f6f380a048c", "71b19c4a-4390-426c-bbe0-77f214a90cfc"],
    "competitors": ["user1", "user2", "user3", "user4"]
}'
```

5. Delete Competition Config
```bash
curl -X DELETE https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-config/134c1f79-a276-493b-87c2-c63f79c3dae1 \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json"
```

Notes:
- All IDs (primaryOwner, auxilaryOwners, eventIds) use UUID format
- Authorization header removed as it's handled by API Gateway
- Rounds field removed from create/update as it will be managed separately
- All endpoints require Content-Type header
```

## Competition Rounds

1. Create Competition Round
```bash
curl -X PUT https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-round/82d44403-9c27-4210-95d5-f5fde2f5671e \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json" \
-d '[
    {
        "competitionId": "1a7e4d2f-2450-41d1-af03-8b3aa714fe69",
        "roundNumber": 1,
        "eventId": "12352e94-b34d-4ee7-a9d1-f1c8e404dec0",
        "roundName": "Comp1 Quarter Finals Match 1",
        "competitorA": "000000000000000000",
        "competitorAScore": 0,
        "competitorB": "333333333333333333",
        "competitorBScore": 0,
        "match": "000000000000000000_333333333333333333",
        "status": "PENDING",
        "isPending": "true",
        "isVotingOpen": "false"
    },
    {
        "competitionId": "1a7e4d2f-2450-41d1-af03-8b3aa714fe69",
        "roundNumber": 2,
        "eventId": "22352e94-b34d-4ee7-a9d1-f1c8e404dec0",
        "roundName": "Comp1 Quarter Finals Match 1",
        "competitorA": "000000000000000000",
        "competitorAScore": 0,
        "competitorB": "444444444444444444",
        "competitorBScore": 0,
        "match": "000000000000000000_444444444444444444",
        "status": "PENDING",
        "isPending": "true",
        "isVotingOpen": "false"
    },
    {
        "competitionId": "2d66c032-f2a2-469b-b578-209437d862a9",
        "roundNumber": 1,
        "eventId": "32352e94-b34d-4ee7-a9d1-f1c8e404dec0",
        "roundName": "Comp2  Quarter Finals Match 1",
        "competitorA": "000000000000000000",
        "competitorAScore": 0,
        "competitorB": "444444444444444444",
        "competitorBScore": 0,
        "match": "000000000000000000_444444444444444444",
        "status": "PENDING",
        "isPending": "true",
        "isVotingOpen": "false"
    }
]'
```
Note: matchup is generated internally to be used by partiql 
    removed the competitors array field for now because we only need compA and compB for now
`<competitorA>_<competitorB> - userId`


2. Get All Rounds for a Competition
```bash
curl -X GET https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-round/competition/1a7e4d2f-2450-41d1-af03-8b3aa714fe69 \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json"
```

3. Get Single Round by Primary Key
```bash
curl -X GET https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-round/1a7e4d2f-2450-41d1-af03-8b3aa714fe69/1 \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json"
```

4. Get competition by event ID
```bash
curl -X GET https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-round/event/12352e94-b34d-4ee7-a9d1-f1c8e404dec0 \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json"
```

5. Update Competition Round
```bash
curl -X PUT https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-round/1a7e4d2f-2450-41d1-af03-8b3aa714fe69 \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json" \
-d '[
        {
            "competitionId": "1a7e4d2f-2450-41d1-af03-8b3aa714fe69",
            "roundName": "Quarter Finals Match 1 - Complete",
            "status": "COMPLETE",
            "eventId": "12352e94-b34d-4ee7-a9d1-f1c8e404dec0",
            "roundNumber": 1,
            "competitorA": "111111111111111111",
            "competitorAScore": 85,
            "competitorB": "222222222222222222",
            "matchup": "111111111111111111_222222222222222222",
            "competitorBScore": 92,
            "isPending": "false",
            "isVotingOpen": "false"
        }
]'
```

6. Delete Competition Round
```bash
curl -X DELETE https://2ykfh7afah.execute-api.us-east-1.amazonaws.com/api/competition-round/1a7e4d2f-2450-41d1-af03-8b3aa714fe69/1 \
-H "Authorization: Bearer nMa54loFfoTGJ6_CSsz4RJcKLtEf1grKAcrkZvijLQB43QgmGtEa6bu-uAOXGPqVx6o3nWiK0gJtBR21yTEEhNIQ1xyBB_4gwO3DBndq" \
-H "Content-Type: application/json"
```

Notes:
- URL Parameters:
  - `primary_owner`: 18 digit userId of the competition owner
  - `competition_id`: UUID of the competition
  - `round_number`: Integer representing the round number
- Status values: "ACTIVE", "COMPLETE", "CANCELLED", "PENDING"
- competitors field should be a JSON string array
- isPending and isVotingOpen are string values ("true"/"false")
- competitorA and competitorB should be valid userId that is 18 digits but as string
- matchup is automatically generated as "<competitorA_userId>_<competitorB_userId>"
- All endpoints require Content-Type header
- Authorization header required for all endpoints except as noted

