package mdns

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAppliesZeroConfigDefaults(t *testing.T) {
	config := New()

	assert.Equal(t, "_http._tcp", config.serviceType)
	assert.NotEmpty(t, config.instanceName)
	assert.True(t, strings.HasSuffix(config.hostname, "."+localDomain),
		"hostname %q should end with .%s", config.hostname, localDomain)
	assert.Nil(t, config.interfaces)
	assert.Equal(t, map[string]string{"path": "/"}, config.text)
}

func TestChainableSetters(t *testing.T) {
	text := map[string]string{"path": "/api"}
	config := New().
		InstanceName("My App").
		ServiceType("_ipp._tcp").
		Hostname("custom.local").
		Interfaces("en0", "en1").
		Text(text)

	assert.Equal(t, "My App", config.instanceName)
	assert.Equal(t, "_ipp._tcp", config.serviceType)
	assert.Equal(t, "custom.local", config.hostname)
	assert.Equal(t, []string{"en0", "en1"}, config.interfaces)
	assert.Equal(t, text, config.text)
}

func TestSettersReturnSameConfigForChaining(t *testing.T) {
	config := New()

	assert.Same(t, config, config.InstanceName("x"))
	assert.Same(t, config, config.ServiceType("_http._tcp"))
	assert.Same(t, config, config.Hostname("x"))
	assert.Same(t, config, config.Interfaces("en0"))
	assert.Same(t, config, config.Text(nil))
}

func TestValidateServiceType(t *testing.T) {
	valid := []string{"_http._tcp", "_ipp._tcp", "_my-service._tcp"}
	for _, serviceType := range valid {
		assert.NoError(t, validateServiceType(serviceType), "expected %q to be valid", serviceType)
	}

	invalid := []string{
		"",            // empty
		"http._tcp",   // missing leading underscore
		"_http._udp",  // wrong transport
		"_http",       // missing transport label
		"_http._tcp.", // trailing dot makes three parts
		"_._tcp",      // empty service label
		"_http.tcp",   // transport missing underscore
	}
	for _, serviceType := range invalid {
		assert.Error(t, validateServiceType(serviceType), "expected %q to be invalid", serviceType)
	}
}

func TestValidateText(t *testing.T) {
	assert.NoError(t, validateText(map[string]string{"path": "/"}))
	assert.NoError(t, validateText(nil))

	// Empty key is rejected.
	assert.Error(t, validateText(map[string]string{"": "value"}))

	// A key=value string of exactly 255 bytes is allowed; 256 is not.
	atLimit := strings.Repeat("a", maxTxtEntryBytes-len("k="))
	assert.NoError(t, validateText(map[string]string{"k": atLimit}))

	overLimit := strings.Repeat("a", maxTxtEntryBytes-len("k=")+1)
	assert.Error(t, validateText(map[string]string{"k": overLimit}))
}

func TestAdvertisementMessage(t *testing.T) {
	msg := New().
		InstanceName("govalin-app").
		ServiceType("_http._tcp").
		Hostname("myhost.local").
		advertisementMessage(7654)

	assert.Contains(t, msg, "govalin-app._http._tcp.local")
	assert.Contains(t, msg, "myhost.local:7654")
	assert.Contains(t, msg, "dns-sd -B _http._tcp")
}

func TestNormalizeHost(t *testing.T) {
	assert.Equal(t, "myhost", normalizeHost("myhost"))
	assert.Equal(t, "myhost", normalizeHost("myhost.local"))
	assert.Equal(t, "myhost", normalizeHost("myhost.local."))
}
