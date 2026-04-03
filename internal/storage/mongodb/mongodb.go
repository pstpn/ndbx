package mongodb

import "go.mongodb.org/mongo-driver/bson/primitive"

func idAlternatives(id string) []any {
	ids := []any{id}
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		ids = append(ids, oid)
	}
	return ids
}

func stringifyID(v any) string {
	switch id := v.(type) {
	case primitive.ObjectID:
		return id.Hex()
	case string:
		return id
	default:
		return ""
	}
}
