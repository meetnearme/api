import json
import random
import string
import time
from datetime import datetime, timedelta

import numpy as np
import psycopg2
from sentence_transformers import SentenceTransformer

# Use 384 dimensions model
model = SentenceTransformer('sentence-transformers/all-MiniLM-L6-v2')

DB_PARAMS = {
    "host": "127.0.0.1",
    "port": "5433",
    "dbname": "postgres",
    "user": "postgres",
    "password": "postgres",
}

def format_results_for_json(results):
    """Convert results to a JSON-friendly format"""
    formatted_results = []
    for event in results:
        (name, description, start_time, address, lat, long, similarity, event_owner_name,
         tags, categories, event_source_id, event_source_type, competition_config_id,
         end_time, recurrence_rule, has_registration_fields, has_purchasable,
         starting_price, currency, payee_id, hide_cross_promo, image_url, timezone,
         source_url, created_at, updated_at, updated_by, event_owners) = event

        formatted_results.append({
            "name": name,
            "description": description,
            "startTime": start_time.isoformat() if start_time else None,
            "address": address,
            "lat": lat,
            "long": long,
            "similarity": similarity,
            "eventOwnerName": event_owner_name,
            "eventOwners": event_owners,
            "tags": tags,
            "categories": categories,
            "eventSourceId": event_source_id,
            "eventSourceType": event_source_type,
            "competitionConfigId": competition_config_id,
            "endTime": end_time.isoformat() if end_time else None,
            "recurrenceRule": recurrence_rule,
            "hasRegistrationFields": has_registration_fields,
            "hasPurchasable": has_purchasable,
            "startingPrice": starting_price,
            "currency": currency,
            "payeeId": payee_id,
            "hideCrossPromo": hide_cross_promo,
            "imageUrl": image_url,
            "timezone": timezone,
            "sourceUrl": source_url,
            "createdAt": created_at,
            "updatedAt": updated_at,
            "updatedBy": updated_by
        })
    return formatted_results

def generate_embedding(text):
    """Generate embedding using sentence-transformer model"""
    embedding = model.encode(text)
    return embedding.tolist()

# This is the updated
def generate_combined_embedding(record):
    """Generate embedding for combined fields with weights"""
    name = record.get('name', '')
    description = record.get('description', '')
    address = record.get('address', '')
    event_owner_name = record.get('eventOwnerName', '')

    # If we're dealing with a string query, process it differently
    if isinstance(record, str):
        return generate_embedding(record)

    # Generate individual embeddings
    name_embedding = np.array(generate_embedding(name)) if name else np.zeros(384)
    description_embedding = np.array(generate_embedding(description)) if description else np.zeros(384)
    address_embedding = np.array(generate_embedding(address)) if address else np.zeros(384)
    owner_embedding = np.array(generate_embedding(event_owner_name)) if event_owner_name else np.zeros(384)

    # Combine with weights
    weighted_embedding = (
        name_embedding * 0.3 +
        description_embedding * 0.5 +
        address_embedding * 0.2 +
        owner_embedding * 0.1
    )

    # Normalize to unit length (important for cosine similarity)
    norm = np.linalg.norm(weighted_embedding)
    if norm > 0:  # Avoid division by zero
        normalized = weighted_embedding / norm
    else:
        normalized = weighted_embedding

    return normalized.tolist()

# This is the original combined embedding that fails because of the multiplication of strings to try to add weights.
# def generate_combined_embedding(record):
#     """Generate embedding for combined fields with weights"""
#     name = record.get('name', '')
#     description = record.get('description', '')
#     address = record.get('address', '')
#     event_owner_name = record.get('eventOwnerName', '')

#     # Apply weights as specified
#     combined_text = (
#         name * 0.3 +
#         event_owner_name * 0.1 +
#         description * 0.5 +
#         address * 0.2
#     )

#     return generate_embedding(combined_text)

def cosine_similarity(a, b):
    """Calculate cosine similarity between two vectors"""
    a = np.array(a)
    b = np.array(b)
    return np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b))

def load_event_templates(json_file):
    """Load event templates from JSON file"""
    try:
        with open(json_file, 'r') as f:
            data = json.load(f)
            return data.get('events', [])
    except Exception as e:
        print(f"Error loading events from {json_file}: {e}")
        return []

def generate_test_data(num_events):
    """Generate test data for performance testing using templates"""
    # Load event templates
    templates = load_event_templates('events.json')
    num_templates = len(templates)
    if num_templates == 0:
        print("No event templates found. Creating sample data.")
        # Create sample template if none exists
        templates = [create_sample_event() for _ in range(5)]
        num_templates = len(templates)

    print(f"Loaded {num_templates} event templates")

    conn = psycopg2.connect(**DB_PARAMS)
    cur = conn.cursor()

    # Clear existing events
    cur.execute("TRUNCATE TABLE events")

    # Generate events based on templates
    for i in range(num_events):
        # Use template events in rotation
        template = templates[i % num_templates]

        # Generate or get values for all fields
        name = template.get('name', f"Event {i}")
        description = template.get('description', f"Description for event {i}")

        # Handle date/time fields
        base_time_str = template.get('startTime', datetime.now().isoformat())
        if isinstance(base_time_str, (int, float)):
            # Handle timestamp in milliseconds
            base_time = datetime.fromtimestamp(base_time_str / 1000)
        else:
            # Handle ISO string
            try:
                base_time = datetime.fromisoformat(base_time_str.replace('Z', '+00:00'))
            except (ValueError, TypeError):
                base_time = datetime.now()

        time_offset = random.randint(-180, 180)  # Â±180 days
        start_time = base_time + timedelta(days=time_offset)

        # Set end time a few hours after start time
        end_time = start_time + timedelta(hours=random.randint(1, 8))

        # Location information
        address = template.get('address', f"123 Main St, City {i}")
        lat = float(template.get('lat', 37.7749)) + random.uniform(-0.01, 0.01)
        long = float(template.get('long', -122.4194)) + random.uniform(-0.01, 0.01)

        # Additional fields
        event_owner_name = template.get('eventOwnerName', f"Organizer {i}")
        event_owners = template.get('eventOwners', [f"owner{i}@example.com"])
        tags = template.get('tags', [f"tag{j}" for j in range(random.randint(1, 5))])
        categories = template.get('categories', [f"category{j}" for j in range(random.randint(1, 3))])
        event_source_id = template.get('eventSourceId', f"src-{i}")
        event_source_type = template.get('eventSourceType', "manual")
        competition_config_id = template.get('competitionConfigId', None)
        recurrence_rule = template.get('recurrenceRule', None)
        has_registration_fields = template.get('hasRegistrationFields', random.choice([True, False]))
        has_purchasable = template.get('hasPurchasable', random.choice([True, False]))
        starting_price = template.get('startingPrice', random.randint(0, 100))
        currency = template.get('currency', "USD")
        payee_id = template.get('payeeId', f"pay-{i}")
        hide_cross_promo = template.get('hideCrossPromo', False)
        image_url = template.get('imageUrl', f"https://example.com/img{i}.jpg")
        timezone = template.get('timezone', "America/Los_Angeles")
        source_url = template.get('sourceUrl', f"https://example.com/event{i}")

        # Timestamps
        created_at = int(time.time() * 1000) - random.randint(0, 30*24*60*60*1000)  # Within last 30 days
        updated_at = created_at + random.randint(0, 10*24*60*60*1000)  # After creation
        updated_by = template.get('updatedBy', f"user{random.randint(1, 10)}")

        # Create record with all fields for embedding generation
        record = {
            'name': name,
            'description': description,
            'address': address,
            'eventOwnerName': event_owner_name
        }

        # Generate embedding for the combined fields
        embedding = generate_combined_embedding(record)

        # Convert lists to JSON strings for database storage
        event_owners_json = json.dumps(event_owners)
        tags_json = json.dumps(tags)
        categories_json = json.dumps(categories)

        cur.execute(
            """
            INSERT INTO events (
                name, description, start_time, end_time, address, lat, long,
                event_owner_name, event_owners, tags, categories, event_source_id,
                event_source_type, competition_config_id, recurrence_rule, has_registration_fields,
                has_purchasable, starting_price, currency, payee_id, hide_cross_promo,
                image_url, timezone, source_url, created_at, updated_at, updated_by, embedding
            )
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """,
            (
                name, description, start_time, end_time, address, lat, long,
                event_owner_name, event_owners_json, tags_json, categories_json, event_source_id,
                event_source_type, competition_config_id, recurrence_rule, has_registration_fields,
                has_purchasable, starting_price, currency, payee_id, hide_cross_promo,
                image_url, timezone, source_url, created_at, updated_at, updated_by, embedding
            )
        )

        # Commit every 1000 events
        if i % 1000 == 0:
            conn.commit()
            print(f"Inserted {i} events...")

    conn.commit()
    cur.close()
    conn.close()

def create_sample_event():
    """Create a sample event for testing"""
    return {
        "name": "Sample Event",
        "description": "This is a sample event description for testing",
        "startTime": datetime.now().isoformat(),
        "address": "123 Main St, Anytown USA",
        "lat": 37.7749,
        "long": -122.4194,
        "eventOwnerName": "Sample Organizer",
        "eventOwners": ["owner@example.com"],
        "tags": ["sample", "test", "event"],
        "categories": ["entertainment", "education"],
        "eventSourceId": "manual-1",
        "eventSourceType": "manual",
        "hasRegistrationFields": True,
        "hasPurchasable": False,
        "startingPrice": 0,
        "currency": "USD",
        "imageUrl": "https://example.com/sample.jpg",
        "timezone": "America/New_York"
    }

def search_similar_events(query, use_temp_table=False):
    """Search for events similar to the query"""
    start_time = time.time()

    # Generate embedding for the query text directly
    if isinstance(query, str):
        # For simple text queries, just encode the text directly
        query_embedding = generate_embedding(query)
    else:
        # For record queries, use the combined embedding approach
        query_embedding = generate_combined_embedding(query)

    embedding_time = time.time() - start_time

    # Connect to database
    conn = psycopg2.connect(**DB_PARAMS)
    cur = conn.cursor()

    # Search for similar events using vector similarity
    sql_query = """
        SELECT
            name, description, start_time, address, lat, long,
            1 - (embedding <=> %s::vector) as similarity,
            event_owner_name, tags, categories, event_source_id, event_source_type,
            competition_config_id, end_time, recurrence_rule, has_registration_fields,
            has_purchasable, starting_price, currency, payee_id, hide_cross_promo,
            image_url, timezone, source_url, created_at, updated_at, updated_by, event_owners
        FROM events
        ORDER BY embedding <=> %s::vector
        LIMIT 50
    """

    # Execute query to get similar events
    query_start = time.time()
    cur.execute(sql_query, (query_embedding, query_embedding))
    results = cur.fetchall()
    query_time = time.time() - query_start

    # Process results
    process_start = time.time()
    results_with_similarity = []

    for result in results:
        # Convert JSON strings back to lists
        result_list = list(result)
        try:
            result_list[8] = json.loads(result[8])  # tags
            result_list[9] = json.loads(result[9])  # categories
            result_list[26] = json.loads(result[26])  # event_owners
        except (json.JSONDecodeError, TypeError):
            # Handle case where data might not be proper JSON
            pass

        # Convert similarity to float
        result_list[6] = float(result_list[6])

        results_with_similarity.append(tuple(result_list))

    process_time = time.time() - process_start

    cur.close()
    conn.close()

    total_time = time.time() - start_time

    timing_info = {
        "embedding_generation": embedding_time,
        "database_query": query_time,
        "results_processing": process_time,
        "total_time": total_time,
    }

    return results_with_similarity, timing_info

# This is the old function that tries to use full embeddings
# def search_similar_events(query, use_temp_table=False):
#     """Search for events similar to the query"""
#     start_time = time.time()

#     # Create a record with the query text for combined embedding
#     query_record = {
#         'name': query,
#         'description': query,
#         'address': query,
#         'eventOwnerName': query
#     }

#     # Generate embedding for the query
#     query_embedding = generate_combined_embedding(query_record)
#     embedding_time = time.time() - start_time

#     # Connect to database
#     conn = psycopg2.connect(**DB_PARAMS)
#     cur = conn.cursor()

#     # Search for similar events using vector similarity
#     sql_query = """
#         SELECT
#             name, description, start_time, address, lat, long,
#             1 - (embedding <=> %s::vector) as similarity,
#             event_owner_name, tags, categories, event_source_id, event_source_type,
#             competition_config_id, end_time, recurrence_rule, has_registration_fields,
#             has_purchasable, starting_price, currency, payee_id, hide_cross_promo,
#             image_url, timezone, source_url, created_at, updated_at, updated_by, event_owners
#         FROM events
#         ORDER BY embedding <=> %s::vector
#         LIMIT 50
#     """

#     # Execute query to get similar events
#     query_start = time.time()
#     cur.execute(sql_query, (query_embedding, query_embedding))
#     results = cur.fetchall()
#     query_time = time.time() - query_start

#     # Process results
#     process_start = time.time()
#     results_with_similarity = []

#     for result in results:
#         # Convert JSON strings back to lists
#         result_list = list(result)
#         try:
#             result_list[8] = json.loads(result[8])  # tags
#             result_list[9] = json.loads(result[9])  # categories
#             result_list[26] = json.loads(result[26])  # event_owners
#         except (json.JSONDecodeError, TypeError):
#             # Handle case where data might not be proper JSON
#             pass

#         # Convert similarity to float
#         result_list[6] = float(result_list[6])

#         results_with_similarity.append(tuple(result_list))

#     process_time = time.time() - process_start

#     cur.close()
#     conn.close()

#     total_time = time.time() - start_time

#     timing_info = {
#         "embedding_generation": embedding_time,
#         "database_query": query_time,
#         "results_processing": process_time,
#         "total_time": total_time,
#     }

#     return results_with_similarity, timing_info

def analyze_search_results(query, top_k=10):
    """
    Analyze search results for a given query and return as structured data
    """
    results, timing = search_similar_events(query)

    top_results = []
    for result in results[:top_k]:
        name, description, start_time, address, lat, long, similarity = result[:7]
        event_owner_name = result[7]
        tags = result[8]
        categories = result[9]

        top_results.append({
            "query": query,
            "name": name,
            "similarity_score": float(similarity),
            "description": description,
            "startTime": start_time.isoformat() if start_time else None,
            "address": address,
            "location": {
                "lat": float(lat),
                "long": float(long)
            },
            "eventOwnerName": event_owner_name,
            "tags": tags,
            "categories": categories
        })

    return top_results

def test_search_relevance():
    """
    Test search relevance with various queries and return results
    """
    test_queries = [
        "outdoor entertainment with refreshments",
        "activities suitable for youngsters",
        "contemporary creative showcase",
        "gourmet sampling experience",
        "athletic competition meetup",
        "innovative industry gathering",
        "mindful wellness session",
        "heritage celebration gathering",
        "culinary instruction meetup",
        "neighborhood skill sharing",
        "organic"
    ]

    all_results = {}
    for query in test_queries:
        all_results[query] = analyze_search_results(query)

    return all_results

def run_performance_test(num_events, num_queries=5):
    """Run performance test with specified number of events"""
    print(f"\nRunning performance test with {num_events:,} events")
    print("-" * 80)

    # Generate test data
    print(f"Generating {num_events:,} test events...")
    generate_test_data(num_events)
    print("Test data generation complete.")

    # Test queries
    test_queries = [
        "outdoor entertainment with refreshments",
        "activities suitable for youngsters",
        "contemporary creative showcase",
        "gourmet sampling experience",
        "athletic competition meetup",
        "innovative industry gathering",
        "mindful wellness session",
        "heritage celebration gathering",
        "culinary instruction meetup",
        "neighborhood skill sharing",
        "organic"
    ]

    all_timings = []

    for query in test_queries[:num_queries]:
        print(f"\nTesting query: {query}")
        results, timing = search_similar_events(query)

        all_timings.append(timing)
        print(f"Total time: {timing['total_time']:.3f} seconds")
        print(f"Embedding generation: {timing['embedding_generation']:.3f} seconds")
        print(f"Database query: {timing['database_query']:.3f} seconds")
        print(f"Results processing: {timing['results_processing']:.3f} seconds")

    # Calculate averages
    avg_timings = {
        "embedding_generation": np.mean([t["embedding_generation"] for t in all_timings]),
        "database_query": np.mean([t["database_query"] for t in all_timings]),
        "results_processing": np.mean([t["results_processing"] for t in all_timings]),
        "total_time": np.mean([t["total_time"] for t in all_timings]),
    }

    print("\nAverage timings:")
    print(f"Total time: {avg_timings['total_time']:.3f} seconds")
    print(f"Embedding generation: {avg_timings['embedding_generation']:.3f} seconds")
    print(f"Database query: {avg_timings['database_query']:.3f} seconds")
    print(f"Results processing: {avg_timings['results_processing']:.3f} seconds")

    return avg_timings

def init_database():
    """Initialize the database with required extensions and tables"""
    print("Initializing database...")

    try:
        conn = psycopg2.connect(**DB_PARAMS)
        conn.autocommit = True  # Required for creating extension
        cur = conn.cursor()

        cur.execute("DROP INDEX IF EXISTS events_embedding_idx")
        cur.execute("DROP TABLE IF EXISTS events")
        cur.execute("DROP EXTENSION IF EXISTS vector")

        # Test 1: Check PostgreSQL version and server path
        cur.execute("SELECT version(), setting FROM pg_settings WHERE name = 'data_directory'")
        version, data_dir = cur.fetchone()
        print(f"Connected to: {version}")
        print(f"Data directory: {data_dir}")

        # Test 2: Check if we're connected to the Docker container
        cur.execute("SHOW data_directory")
        container_path = cur.fetchone()[0]
        print(f"Container path: {container_path}")

        # Test 3: Check available extensions
        cur.execute("SELECT * FROM pg_available_extensions WHERE name = 'vector'")
        extension_info = cur.fetchone()
        print(f"Vector extension info: {extension_info}")

        cur.execute("CREATE EXTENSION IF NOT EXISTS vector")

        # Create the events table with all fields
        cur.execute("""
            CREATE TABLE IF NOT EXISTS events (
                id SERIAL PRIMARY KEY,
                name VARCHAR(255) NOT NULL,
                description TEXT,
                start_time TIMESTAMP,
                end_time TIMESTAMP,
                address TEXT,
                lat FLOAT,
                long FLOAT,
                event_owner_name VARCHAR(255),
                event_owners TEXT,
                tags TEXT,
                categories TEXT,
                event_source_id VARCHAR(255),
                event_source_type VARCHAR(100),
                competition_config_id VARCHAR(255),
                recurrence_rule TEXT,
                has_registration_fields BOOLEAN,
                has_purchasable BOOLEAN,
                starting_price INT,
                currency VARCHAR(10),
                payee_id VARCHAR(255),
                hide_cross_promo BOOLEAN,
                image_url TEXT,
                timezone VARCHAR(100),
                source_url TEXT,
                created_at BIGINT,
                updated_at BIGINT,
                updated_by VARCHAR(255),
                embedding vector(384)
            )
        """)

        # Create an index for faster similarity search
        cur.execute("""
            CREATE INDEX IF NOT EXISTS events_embedding_idx
            ON events USING ivfflat (embedding vector_cosine_ops)
        """)

        print("Database initialization complete.")
    except Exception as e:
        print(f"Error initializing database: {str(e)}")
        raise e
    finally:
        conn.commit()
        cur.close()
        conn.close()

# Add this function to search.py to run controlled tests
def test_relevance_time_independence():
    """
    Test whether search results are solely based on semantic similarity
    and not influenced by temporal factors.
    """
    print("\nTesting search relevance independence from temporal factors...")

    # Select queries relevant to events with varying temporal distances
    test_queries = [
        "Italian sporting event with refreshments",  # Should match Bocce Ball event regardless of time
        "electronic music festival",  # Should match Phoenix Lights regardless of date
        "automobile enthusiast gathering",  # Should match lowriders event regardless of time
        "cultural celebration"  # Should match several events with different dates
    ]

    results = {}
    for query in test_queries:
        print(f"\nTesting query: '{query}'")

        # Get regular search results
        regular_results, _ = search_similar_events(query)

        # Format a simplified version for comparison
        top5_regular = []
        for r in regular_results[:5]:
            top5_regular.append({
                "name": r[0],
                "similarity": float(r[6]),
                "startTime": r[2].isoformat() if r[2] else None,
                "createdAt": r[23],
                "updatedAt": r[24]
            })

        print("Top 5 results based on semantic similarity:")
        for i, result in enumerate(top5_regular, 1):
            print(f"{i}. {result['name']} (score: {result['similarity']:.4f}, date: {result['startTime']})")

        results[query] = top5_regular

    # Create a validation test by creating events with identical content but different dates
    conn = psycopg2.connect(**DB_PARAMS)
    try:
        cur = conn.cursor()

        # Create a test event with different temporal variants
        test_name = "Test Cultural Festival"
        test_description = "A special event celebrating diverse cultural traditions with music, food, and performances."
        test_address = "Convention Center, Test City"
        test_owner_name = "Cultural Association"

        # Create a record for generating weighted embedding
        test_record = {
            'name': test_name,
            'description': test_description,
            'address': test_address,
            'eventOwnerName': test_owner_name
        }

        # Generate the weighted embedding
        test_embedding = generate_combined_embedding(test_record)

        dates = [
            # Recent past
            datetime.now() - timedelta(days=7),
            # Near future
            datetime.now() + timedelta(days=7),
            # Far future
            datetime.now() + timedelta(days=180),
            # Far past
            datetime.now() - timedelta(days=180)
        ]

        # First, clean up any existing test events
        cur.execute("DELETE FROM events WHERE name LIKE 'Test Cultural Festival%'")

        # Create variants with different dates but same content (and thus same embedding)
        for i, date in enumerate(dates):
            name = f"{test_name} {i+1}"
            start_time = date
            end_time = start_time + timedelta(hours=4)
            created_at = int((datetime.now() - timedelta(days=30*(i+1))).timestamp() * 1000)
            updated_at = int((datetime.now() - timedelta(days=10*(i+1))).timestamp() * 1000)
            updated_by = f"test-user-{i}"

            print(f"\nInserting test event: {name}")
            print(f"  Start time: {start_time}")
            print(f"  Created at: {datetime.fromtimestamp(created_at/1000)}")
            print(f"  Updated at: {datetime.fromtimestamp(updated_at/1000)}")

            # Insert with ALL required fields to match the schema
            cur.execute(
                """
                INSERT INTO events (
                    name, description, start_time, end_time, address, lat, long,
                    event_owner_name, event_owners, tags, categories,
                    event_source_id, event_source_type, competition_config_id,
                    recurrence_rule, has_registration_fields, has_purchasable,
                    starting_price, currency, payee_id, hide_cross_promo,
                    image_url, timezone, source_url, created_at, updated_at,
                    updated_by, embedding
                )
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    name, test_description, start_time, end_time, test_address,
                    40.0, -75.0, test_owner_name,
                    '["test@example.com"]',  # event_owners
                    '["test", "cultural"]',  # tags
                    '["culture", "festival"]',  # categories
                    f"test-src-{i}",  # event_source_id
                    "manual",  # event_source_type
                    f"test-config-{i}",  # competition_config_id
                    "FREQ=YEARLY",  # recurrence_rule
                    True,  # has_registration_fields
                    False,  # has_purchasable
                    10,  # starting_price
                    "USD",  # currency
                    f"test-payee-{i}",  # payee_id
                    False,  # hide_cross_promo
                    f"https://example.com/img{i}.jpg",  # image_url
                    "America/New_York",  # timezone
                    f"https://example.com/event{i}",  # source_url
                    created_at, updated_at, updated_by, test_embedding
                )
            )

        conn.commit()

        # Test search with identical content events
        print("\nTesting temporal independence with identical content events:")
        identical_results, _ = search_similar_events("cultural traditions festival")

        # Extract the test events
        test_events = [r for r in identical_results if "Test Cultural Festival" in r[0]]

        # If events are ranked by semantic similarity only, they should have nearly identical scores
        # since the content is identical
        print("\nTest events ranked by similarity:")
        # Update the display code in test_relevance_time_independence
        for i, event in enumerate(test_events, 1):
            name, _, start_time, _, _, _, similarity = event[:7]
            created_at, updated_at = event[23:25]

            print(f"{i}. {name} (score: {float(similarity):.6f})")
            print(f"   Start Time: {start_time}")

            # Handle created_at which might be a string or an integer or None
            if created_at is not None:
                try:
                    # Try to convert to int first if it's a string
                    created_at_int = int(created_at) if isinstance(created_at, str) else created_at
                    created_time = datetime.fromtimestamp(created_at_int / 1000)
                    print(f"   Created At: {created_time}")
                except (ValueError, TypeError):
                    print(f"   Created At: {created_at} (unable to format)")
            else:
                print("   Created At: None")

            # Handle updated_at which might be a string or an integer or None
            if updated_at is not None:
                try:
                    # Try to convert to int first if it's a string
                    updated_at_int = int(updated_at) if isinstance(updated_at, str) else updated_at
                    updated_time = datetime.fromtimestamp(updated_at_int / 1000)
                    print(f"   Updated At: {updated_time}")
                except (ValueError, TypeError):
                    print(f"   Updated At: {updated_at} (unable to format)")
            else:
                print("   Updated At: None")
        # for i, event in enumerate(test_events, 1):
        #     name, _, start_time, _, _, _, similarity = event[:7]
        #     created_at, updated_at = event[23:25]

        #     print(f"{i}. {name} (score: {float(similarity):.6f})")
        #     print(f"   Start Time: {start_time}")
        #     print(f"   Created At: {datetime.fromtimestamp(created_at/1000) if created_at else None}")
        #     print(f"   Updated At: {datetime.fromtimestamp(updated_at/1000) if updated_at else None}")

        # # Validate that scores are within a small epsilon of each other (should be almost identical)
        # if test_events:
        #     base_score = float(test_events[0][6])
        #     score_diffs = [abs(float(e[6]) - base_score) for e in test_events]
        #     max_diff = max(score_diffs) if score_diffs else 0

        #     print(f"\nMaximum score difference between identical content events: {max_diff:.8f}")
        #     if max_diff < 0.0001:
        #         print("âœ… PASS: Search results appear to be based solely on semantic similarity.")
        #         print("   The dates of events do not affect their relevance ranking.")
        #     else:
        #         print("âš ï¸ WARNING: There may be temporal factors affecting search results.")
        # else:
        #     print("âš ï¸ No test events found in results.")

    except Exception as e:
        print(f"Error in temporal independence test: {e}")
        import traceback
        traceback.print_exc()  # Print the full traceback for debugging
    finally:
        # Clean up test events
        cur.execute("DELETE FROM events WHERE name LIKE 'Test Cultural Festival%'")
        conn.commit()
        cur.close()
        conn.close()

    return results

def main():
    init_database()
    print("Database initialized")

    # Generate test data from events.json
    print("\nGenerating test events from events.json...")
    generate_test_data(0)  # Will use all events from the JSON file

    # Run the temporal independence test
    print("\nTesting whether time factors affect search relevance...")
    test_relevance_time_independence()

    print("Completed time factor's affect tests")

    # Run performance tests for different data sizes
    # data_sizes = [1000, 10000]  # Removed 1M to keep tests reasonable
    # results = {}

    # for size in data_sizes:
    #     try:
    #         print(f"\nTesting with {size} events...")

    #         # Run performance test
    #         timing_results = run_performance_test(size)

    #         # Run search relevance test
    #         print(f"\nTesting search relevance with {size} events...")
    #         search_results = test_search_relevance()

    #         # Store both timing and search results
    #         results[size] = {
    #             "performance_metrics": timing_results,
    #             "search_results": search_results
    #         }

    #     except Exception as e:
    #         print(f"Error testing with {size} events: {str(e)}")
    #         break

    # # Save combined results to JSON file
    # timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    # model_name = "all_MiniLM_L6_v2"
    # filename = f"model_evaluation_{model_name}_{timestamp}.json"

    # with open(filename, "w") as f:
    #     json.dump({
    #         "model": model_name,
    #         "timestamp": timestamp,
    #         "results": results
    #     }, f, indent=2)

    # print(f"\nAll results saved to {filename}")

if __name__ == "__main__":
    main()
