## Users

1. Create User

```bash

# example of all fields
curl -X POST https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/users \
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
 curl -X POST  https://vze2jkkcn5.execute-api.us-east-1.amazonaws.com/api/users \
 -H "Content-Type: application/json" \
 -d '{
     "name": "John Doe",
     "email": "john.doe@example.com",
     "role": "standard_user"
 }'

 
```

2. Get User by ID
```bash
curl -X GET <instance_url>/users/<:id>

curl -X GET https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/users/310266d0-196e-4c72-b10d-97616976f650

```

3. Get Users
```bash
curl -X GET https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/users

```

3. Update user
```bash
curl -X PUT https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/users/c9413f71-bb0e-43c9-bc3a-fafc64c5c799 \
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
curl -X DELETE https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/users/310266d0-196e-4c72-b10d-97616976f650 
```

## Purchasables

1. Create Purchasable
```bash
curl -X POST https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/purchasables \
     -H "Content-Type: application/json" \
     -d '{
           "user_id": "ea49a5f8-e27c-47b0-8237-6f6f380a048c",
           "name": "Sample Item 2",
           "item_type": "ticket",
           "cost": 69.99,
           "currency": "USD",
           "donation_ratio": 0.10,
           "inventory": 100,
           "charge_recurrence_interval": "month",
           "charge_recurrence_interval_count": 1,
           "charge_recurrence_end_date": "2024-12-31T23:59:59Z"
         }'

```

2. Update Purchasable
```bash
curl -X PUT https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/purchasables/a8950a91-f353-4032-8c5d-03335ae2b0d9 \
     -H "Content-Type: application/json" \
     -d '{
           "name": "Updated Item",
           "item_type": "membership",
           "cost": 99.99,
           "currency": "USD",
           "donation_ratio": 0.15,
           "inventory": 200,
           "charge_recurrence_interval": "year",
           "charge_recurrence_interval_count": 1,
           "charge_recurrence_end_date": "2025-12-31T23:59:59Z"
         }'

```
3. Get Purchasable by ID
```bash
curl -X GET https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/purchasables/a8950a91-f353-4032-8c5d-03335ae2b0d9 \
     -H "Content-Type: application/json"

```
4. Get Purchasables by UserID
```bash
curl -X GET https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/purchasables/user/ea49a5f8-e27c-47b0-8237-6f6f380a048c \
     -H "Content-Type: application/json"
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/purchasables/user/40164b69-b2fe-4706-aa12-7d6ec3845c95 \
     -H "Content-Type: application/json"
```
5. Delete Purchasable
```bash
curl -X DELETE https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/purchasables/a8950a91-f353-4032-8c5d-03335ae2b0d9 \
     -H "Content-Type: application/json"

```


## Event RSVPs
1. Create EventRsvp
```bash
curl -X POST https://4ektcpsj01.execute-api.us-east-1.amazonaws.com/api/event-rsvps \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "ea49a5f8-e27c-47b0-8237-6f6f380a048c",
    "event_id": "6ce1be30-f700-475c-b84a-49af0c73f337",
    "event_source_id": "71b19c4a-4390-426c-bbe0-77f214a90cfc",
    "event_source_type": "internalRecurrence",
    "status": "Yes"
}'
```

2. GET EventRsvp By ID
```bash
curl -X GET https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/event-rsvps/b5c93f43-db5f-4c37-870b-0c2c4878e4fb \
  -H "Content-Type: application/json"

```

3. GET EventRsvp By UserID
```bash
curl -X GET https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/event-rsvps/user/ea49a5f8-e27c-47b0-8237-6f6f380a048c \
  -H "Content-Type: application/json"

```
4. GET EventRsvp By EventID
```bash
curl -X GET https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/event-rsvps/event/6ce1be30-f700-475c-b84a-49af0c73f337 \
  -H "Content-Type: application/json"
```

5. Update EventRsvp
```bash
curl -X PUT https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/event-rsvps/b5c93f43-db5f-4c37-870b-0c2c4878e4fb \
  -H "Content-Type: application/json" \
  -d '{
    "status": "Maybe"
}'

```

6. Delete EventRsvp
```bash
curl -X DELETE https://u7euqonjl8.execute-api.us-east-1.amazonaws.com/api/event-rsvps/b5c93f43-db5f-4c37-870b-0c2c4878e4fb \
  -H "Content-Type: application/json"

```

# DynamoDB Endpoints

## Registration Fields
1. Create Registration Fields
```bash
curl -X POST https://ljd19o7833.execute-api.us-east-1.amazonaws.com/api/registration-fields/62352e94-b34d-4ee7-a9d1-f1c8e404dec0 \
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
curl -X GET https://ljd19o7833.execute-api.us-east-1.amazonaws.com/api/registration-fields/62352e94-b34d-4ee7-a9d1-f1c8e404dec0 \
-H "Content-Type: application/json"
```

3. Update Registration Fields
```bash
curl -X PUT https://ljd19o7833.execute-api.us-east-1.amazonaws.com/api/registration-fields/62352e94-b34d-4ee7-a9d1-f1c8e404dec0 \
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
curl -X DELETE https://ljd19o7833.execute-api.us-east-1.amazonaws.com/api/registration-fields/62352e94-b34d-4ee7-a9d1-f1c8e404dec0 \
-H "Content-Type: application/json"
```


## Registrations
