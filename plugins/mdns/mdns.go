// Package mdns provides an opt-in GOVALIN plugin that advertises the running
// server on the local network as a DNS-SD `_http._tcp` service over multicast
// DNS (Bonjour/Zeroconf).
//
// Once registered the app becomes resolvable (`<host>.local`) and browsable
// with zero configuration via Safari's Bonjour list, `dns-sd -B _http._tcp`
// (macOS) or `avahi-browse -a` (Linux). The plugin only advertises — it does
// not browse or discover other services.
//
// The mDNS/brutella dependency is isolated to this package per the plugin
// complexity boundary; GOVALIN core never imports it.
//
// # Failure handling
//
// Failure handling is deliberately asymmetric:
//
//   - Misconfiguration (an invalid or non-`_tcp` service type, oversized TXT
//     entries) is validated in OnInit and is fatal, consistent with the cors
//     plugin.
//   - Runtime registration/broadcast failure (no multicast route, blocked UDP
//     5353, a container without host networking) is non-fatal — the HTTP server
//     keeps serving and an actionable warning is logged.
//
// # Docker
//
// Multicast typically does not cross the default Docker bridge network, so mDNS
// advertisement will silently reach nothing there. Run the container with host
// networking (`docker run --network host`) to let mDNS work.
package mdns

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/brutella/dnssd"
	"github.com/pkkummermo/govalin"
)

const (
	// tcpTransport is the only transport DNS-SD service advertisement supports
	// in this plugin; the service type must end with it.
	tcpTransport = "_tcp"

	// localDomain is the multicast DNS domain all advertised services live in.
	localDomain = "local"

	// maxTxtEntryBytes is the maximum size of a single `key=value` TXT string
	// per RFC 6763 §6.1 (a DNS character-string is length-prefixed by a single
	// byte, so it cannot exceed 255 bytes).
	maxTxtEntryBytes = 255
)

// Config holds the mDNS advertisement configuration. Construct it with New and
// adjust it with the chainable setters before handing it to config.Plugin.
type Config struct {
	instanceName string
	serviceType  string
	hostname     string
	interfaces   []string
	text         map[string]string

	// runtime state, populated in Apply.
	cancel context.CancelFunc
	done   chan struct{}
}

// New creates an mDNS plugin with zero-config defaults:
//
//   - instance name = executable base name
//   - service type  = _http._tcp
//   - hostname      = <os-hostname>.local
//   - interfaces    = all multicast-capable interfaces (auto-detected)
//   - TXT           = {path: /}
//
// Every default is overridable via the chainable setters. New takes no required
// arguments.
func New() *Config {
	return &Config{
		instanceName: defaultInstanceName(),
		serviceType:  "_http._tcp",
		hostname:     defaultHostname(),
		interfaces:   nil,
		text:         map[string]string{"path": "/"},
	}
}

func (config *Config) Name() string {
	return "mDNS plugin"
}

// OnInit validates the configuration (fatal on misconfiguration) and registers
// the shutdown hook that withdraws the service via goodbye packets.
func (config *Config) OnInit(govalinConfig *govalin.Config) {
	config.checkConfiguration()

	govalinConfig.Events(func(events *govalin.ServerEvents) {
		events.AddOnServerShutdown(config.withdraw)
	})
}

// Apply advertises the service on the network. The listener is already bound by
// the time Apply runs, so the bound port is read from App.Port(). Runtime
// failures are non-fatal: they log an actionable warning and leave the HTTP
// server untouched.
func (config *Config) Apply(app *govalin.App) {
	port := app.Port()

	service, serviceErr := dnssd.NewService(dnssd.Config{
		Name:   config.instanceName,
		Type:   config.serviceType,
		Domain: localDomain,
		Host:   normalizeHost(config.hostname),
		Ifaces: config.interfaces,
		Port:   int(port),
		Text:   config.text,
	})
	if serviceErr != nil {
		config.warnRuntimeFailure(serviceErr)
		return
	}

	responder, responderErr := dnssd.NewResponder()
	if responderErr != nil {
		config.warnRuntimeFailure(responderErr)
		return
	}

	if _, addErr := responder.Add(service); addErr != nil {
		config.warnRuntimeFailure(addErr)
		return
	}

	slog.Info(config.advertisementMessage(port))

	ctx, cancel := context.WithCancel(context.Background())
	config.cancel = cancel
	config.done = make(chan struct{})

	go func() {
		defer close(config.done)
		// Respond blocks until the context is cancelled (sending goodbye
		// packets on the way out) or initial registration fails.
		if respondErr := responder.Respond(ctx); respondErr != nil && !errors.Is(respondErr, context.Canceled) {
			config.warnRuntimeFailure(respondErr)
		}
	}()
}

// withdraw cancels the advertisement so the responder sends goodbye packets,
// then waits for them to flush before returning.
func (config *Config) withdraw() {
	if config.cancel == nil {
		return
	}

	config.cancel()
	<-config.done
}

// InstanceName sets the human-readable service instance name shown in discovery
// tooling. Defaults to the executable base name.
func (config *Config) InstanceName(name string) *Config {
	config.instanceName = name
	return config
}

// ServiceType sets the DNS-SD service type in full `_service._tcp` form (e.g.
// "_http._tcp"). The transport is fixed at `_tcp`; any other value is fatal at
// OnInit. Defaults to "_http._tcp".
func (config *Config) ServiceType(serviceType string) *Config {
	config.serviceType = serviceType
	return config
}

// Hostname sets the host the service resolves to. A trailing ".local" is
// optional and normalized away internally. Defaults to "<os-hostname>.local".
func (config *Config) Hostname(hostname string) *Config {
	config.hostname = hostname
	return config
}

// Interfaces restricts advertisement to the named network interfaces (e.g.
// "en0"). Defaults to all multicast-capable interfaces (auto-detected).
func (config *Config) Interfaces(interfaces ...string) *Config {
	config.interfaces = interfaces
	return config
}

// Text sets the TXT record key/value pairs advertised with the service. Each
// `key=value` entry must fit DNS-SD size limits; an oversized entry is fatal at
// OnInit. Defaults to {path: /}.
func (config *Config) Text(text map[string]string) *Config {
	config.text = text
	return config
}

func (config *Config) checkConfiguration() {
	if err := validateServiceType(config.serviceType); err != nil {
		slog.Error(fmt.Sprintf("mDNS plugin: %v. The service type must be in full "+
			"DNS-SD form with a _tcp transport, e.g. \"_http._tcp\".", err))
		os.Exit(1)
	}

	if err := validateText(config.text); err != nil {
		slog.Error(fmt.Sprintf("mDNS plugin: %v", err))
		os.Exit(1)
	}
}

// advertisementMessage describes where the service is advertised and what it
// points to, for the success log line emitted once setup succeeds.
func (config *Config) advertisementMessage(port uint16) string {
	instance := config.instanceName + "." + config.serviceType + "." + localDomain
	host := normalizeHost(config.hostname) + "." + localDomain

	return fmt.Sprintf("mDNS plugin: advertising %q → %s:%d on the local network "+
		"(browse with \"dns-sd -B %s\" or \"avahi-browse -a\")",
		instance, host, port, config.serviceType)
}

func (config *Config) warnRuntimeFailure(err error) {
	slog.Warn(fmt.Sprintf("mDNS plugin: could not advertise %q over multicast DNS: %v. "+
		"The HTTP server is unaffected and keeps serving. This usually means the environment "+
		"has no multicast route or UDP port 5353 is blocked — for example a Docker container "+
		"on the default bridge network. To enable mDNS, run the container with host networking "+
		"(docker run --network host) or otherwise ensure multicast traffic on UDP 5353 can "+
		"reach the LAN.", config.instanceName, err))
}

// validateServiceType ensures the service type is in full `_service._tcp` form
// with a _tcp transport.
func validateServiceType(serviceType string) error {
	parts := strings.Split(serviceType, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid service type %q", serviceType)
	}

	if len(parts[0]) < 2 || !strings.HasPrefix(parts[0], "_") {
		return fmt.Errorf("invalid service type %q: the service label must start with \"_\"", serviceType)
	}

	if parts[1] != tcpTransport {
		return fmt.Errorf("invalid service type %q: transport must be %q", serviceType, tcpTransport)
	}

	return nil
}

// validateText ensures every TXT entry has a non-empty key and fits the DNS-SD
// per-string size limit.
func validateText(text map[string]string) error {
	for key, value := range text {
		if key == "" {
			return errors.New("TXT record contains an empty key, which is not allowed")
		}

		entrySize := len(key) + len("=") + len(value)
		if entrySize > maxTxtEntryBytes {
			return fmt.Errorf("TXT entry %q is %d bytes, exceeding the DNS-SD limit of %d bytes "+
				"per key=value string; shorten the key or value", key, entrySize, maxTxtEntryBytes)
		}
	}

	return nil
}

// normalizeHost strips a trailing ".local" (and any trailing dot) so the
// hostname is a bare label; the multicast domain is supplied separately. This
// lets callers pass either "myhost" or "myhost.local".
func normalizeHost(hostname string) string {
	host := strings.TrimSuffix(hostname, ".")
	host = strings.TrimSuffix(host, "."+localDomain)
	return host
}

func defaultInstanceName() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Base(exe)
	}
	return filepath.Base(os.Args[0])
}

func defaultHostname() string {
	if host, err := os.Hostname(); err == nil {
		return host + "." + localDomain
	}
	return "govalin." + localDomain
}
