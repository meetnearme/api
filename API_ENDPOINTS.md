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



