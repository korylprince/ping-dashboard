package main

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

// Schema represents a yaml schema
type Schema []struct {
	Category string   `json:"category"`
	Hosts    []string `json:"hosts"`
}

func (s Schema) MarshalJSON() ([]byte, error) {
	type schema2 Schema
	type schema struct {
		Type   string  `json:"t"`
		Schema schema2 `json:"s"`
	}

	sch := &schema{Type: "s", Schema: schema2(s)}
	return json.Marshal(sch)
}

// UnmarshalSchema parses and returns a schema from r
func UnmarshalSchema(r io.Reader) (Schema, error) {
	s := make(Schema, 0)
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("could not decode schema: %w", err)
	}

	return s, nil
}
