package qlog

import (
	"github.com/quic-go/quic-go/logging"
	"strings"
)

type Config struct {
	ExcludeEventsByDefault bool
	// keys in form "<category>:<name>"
	// e.g. "transport:packet_received"
	IncludedEvents map[eventKey]bool
	Title          string
	CodeVersion    string
	GroupID        string
	ODCID          string
	VantagePoint   logging.Perspective
}

type eventKey struct {
	Category string
	Name     string
}

func (c *Config) SetIncludedEvents(includedEvents map[string]bool) {
	c.IncludedEvents = map[eventKey]bool{}
	for stringKey, value := range includedEvents {
		parts := strings.Split(stringKey, ":")
		category := parts[0]
		name := parts[1]
		c.IncludedEvents[eventKey{Category: category, Name: name}] = value
	}
}

func (c *Config) Included(category string, name string) bool {
	if included, ok := c.IncludedEvents[eventKey{Category: category, Name: name}]; ok {
		return included
	}
	return !c.ExcludeEventsByDefault
}

func (c *Config) Populate() *Config {
	if c == nil {
		c = &Config{}
	}
	return c
}

func (c *Config) Copy() *Config {
	return &Config{
		ExcludeEventsByDefault: c.ExcludeEventsByDefault,
		IncludedEvents:         c.IncludedEvents,
		Title:                  c.Title,
		CodeVersion:            c.CodeVersion,
		GroupID:                c.GroupID,
		ODCID:                  c.ODCID,
		VantagePoint:           c.VantagePoint,
	}
}
