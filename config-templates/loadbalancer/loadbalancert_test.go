package loadbalancer

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJsonMarshalTest(t *testing.T) {

	var configObject LoadbalancerConfig

	config := "{\"ports\": [{\"port\":80,\"protocol\":\"tcp\"},{\"port\":443,\"protocol\":\"tcp\"},{\"port\":8080,\"protocol\":\"tcp\"},{\"port\":9000,\"protocol\":\"tcp\"},{\"port\":9001,\"protocol\":\"tcp\"}]}"

	unmarshalError := json.Unmarshal([]byte(config), &configObject)

	assert.Nil(t, unmarshalError)

	assert.NotEqual(t, 0, len(configObject.Ports))
}
