package migration

// TransformFunc defines the signature for transformer functions
type TransformFunc func(map[string]interface{}) (map[string]interface{}, error)

// TransformerRegistry holds all available transformers
var TransformerRegistry = map[string]TransformFunc{
    "test_transformer": func(doc map[string]interface{}) (map[string]interface{}, error) {
        // Add a test field to verify transformation
        doc["_test_migration"] = true
        return doc, nil
    },
    "tensor_weights": func(doc map[string]interface{}) (map[string]interface{}, error) {
        // Implement the actual tensor weights transformation
        if name, ok := doc["name"].(string); ok {
            doc["_tensor_weights"] = map[string]float64{
                "name": 0.3,
                "description": 0.5,
                "address": 0.2,
            }
        }
        return doc, nil
    },
}
