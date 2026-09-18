package architect

import "encoding/json"

func ContractJSONSchema() map[string]any {
	defs := schemaDefinitions()
	return schemaRoot("ArchitectureContract", defs, map[string]any{
		"schema_version":       map[string]any{"const": CurrentSchemaVersion, "type": "string"},
		"revision":             map[string]any{"minimum": 1, "type": "integer"},
		"scope":                textSchema(1, 500),
		"architecture_style":   nullable(textSchema(1, 500)),
		"quality_attributes":   arraySchema(ref("#/$defs/QualityAttribute"), 0, 20),
		"components":           arraySchema(ref("#/$defs/Component"), 0, 200),
		"external_boundaries":  arraySchema(ref("#/$defs/ExternalBoundary"), 0, 100),
		"dependency_rules":     arraySchema(ref("#/$defs/DependencyRule"), 0, 500),
		"required_practices":   arraySchema(textSchema(1, 500), 0, 200),
		"prohibited_practices": arraySchema(textSchema(1, 500), 0, 200),
		"decision_ids":         arraySchema(regexSchema("^ADR-[0-9]{3}$"), 0, 200),
		"unresolved_questions": arraySchema(ref("#/$defs/ClarificationQuestion"), 0, 50),
	}, []string{"schema_version", "revision", "scope"})
}

func BundleJSONSchema() map[string]any {
	defs := schemaDefinitions()
	root := map[string]any{
		"contract":        ref("#/$defs/ArchitectureContract"),
		"decisions":       arraySchema(ref("#/$defs/ArchitectureDecisionArtifact"), 1, 200),
		"project_context": textSchema(1, 20_000),
		"coding_handoff":  textSchema(1, 20_000),
	}
	document := schemaRoot("ArchitectureArtifactBundle", defs, root, []string{"contract", "decisions", "project_context", "coding_handoff"})
	document["$defs"].(map[string]any)["ArchitectureContract"] = ContractJSONSchema()
	return document
}

func SchemaJSON(schema map[string]any) ([]byte, error) {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func schemaRoot(title string, defs map[string]any, properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$comment":             "Go-owned canonical schema; contract version " + CurrentSchemaVersion,
		"$defs":                defs,
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
		"title":                title,
		"type":                 "object",
	}
}

func schemaDefinitions() map[string]any {
	return map[string]any{
		"ClarificationQuestion": objectSchema(map[string]any{
			"id":              regexSchema("^Q-[0-9]{3}$"),
			"question":        textSchema(1, 2_000),
			"decision_impact": textSchema(1, 2_000),
			"critical":        map[string]any{"default": false, "type": "boolean"},
			"answer":          nullable(textSchema(1, 20_000)),
		}, []string{"id", "question", "decision_impact"}),
		"QualityAttribute": objectSchema(map[string]any{
			"name":              textSchema(1, 500),
			"priority":          map[string]any{"minimum": 1, "maximum": 5, "type": "integer"},
			"rationale":         textSchema(1, 2_000),
			"measurable_signal": nullable(textSchema(1, 2_000)),
		}, []string{"name", "priority", "rationale"}),
		"Component": objectSchema(map[string]any{
			"id":                regexSchema("^[a-z][a-z0-9-]*$"),
			"responsibility":    textSchema(1, 2_000),
			"owns_data":         arraySchema(textSchema(1, 500), 0, 100),
			"public_interfaces": arraySchema(textSchema(1, 500), 0, 100),
		}, []string{"id", "responsibility"}),
		"ExternalBoundary": objectSchema(map[string]any{
			"id":             regexSchema("^[a-z][a-z0-9-]*$"),
			"responsibility": textSchema(1, 2_000),
		}, []string{"id", "responsibility"}),
		"DependencyRule": objectSchema(map[string]any{
			"source":        regexSchema("^[a-z][a-z0-9-]*$"),
			"target":        regexSchema("^[a-z][a-z0-9-]*$"),
			"policy":        map[string]any{"enum": []string{"allow", "deny", "allow-via-interface"}, "type": "string"},
			"via_interface": nullable(textSchema(1, 500)),
			"rationale":     textSchema(1, 2_000),
		}, []string{"source", "target", "policy", "rationale"}),
		"ArchitectureDecision": objectSchema(map[string]any{
			"id":                    regexSchema("^ADR-[0-9]{3}$"),
			"title":                 textSchema(1, 500),
			"status":                map[string]any{"enum": []string{"proposed", "accepted", "rejected", "superseded"}, "type": "string"},
			"context":               textSchema(1, 20_000),
			"drivers":               arraySchema(textSchema(1, 2_000), 1, 30),
			"considered_option_ids": arraySchema(regexSchema("^OPT-[0-9]{3}$"), 1, 5),
			"selected_option_id":    nullable(regexSchema("^OPT-[0-9]{3}$")),
			"decision":              textSchema(1, 20_000),
			"positive_consequences": arraySchema(textSchema(1, 2_000), 0, 30),
			"negative_consequences": arraySchema(textSchema(1, 2_000), 0, 30),
			"assumptions":           arraySchema(textSchema(1, 2_000), 0, 30),
			"validation_criteria":   arraySchema(textSchema(1, 2_000), 1, 30),
			"supersedes":            arraySchema(regexSchema("^ADR-[0-9]{3}$"), 0, 100),
		}, []string{"id", "title", "status", "context", "drivers", "considered_option_ids", "decision", "validation_criteria"}),
		"ArchitectureDecisionArtifact": objectSchema(map[string]any{
			"schema_version": map[string]any{"const": CurrentSchemaVersion, "type": "string"},
			"revision":       map[string]any{"minimum": 1, "type": "integer"},
			"decision":       ref("#/$defs/ArchitectureDecision"),
		}, []string{"schema_version", "revision", "decision"}),
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{"additionalProperties": false, "properties": properties, "required": required, "type": "object"}
}

func textSchema(min, max int) map[string]any {
	return map[string]any{"maxLength": max, "minLength": min, "type": "string"}
}

func regexSchema(pattern string) map[string]any {
	return map[string]any{"pattern": pattern, "type": "string"}
}

func arraySchema(items map[string]any, min, max int) map[string]any {
	value := map[string]any{"items": items, "maxItems": max, "type": "array"}
	if min > 0 {
		value["minItems"] = min
	}
	return value
}

func nullable(schema map[string]any) map[string]any {
	return map[string]any{"anyOf": []any{schema, map[string]any{"type": "null"}}}
}

func ref(value string) map[string]any { return map[string]any{"$ref": value} }
