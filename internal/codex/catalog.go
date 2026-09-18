package codex

import (
	"path/filepath"
	"strings"
)

const CanonicalReferenceBase = "https://github.com/leomuf/ai-software-architect/blob/main/shared/skills/evaluate-architecture-options/references/"

type ReferenceSpec struct {
	Name     string
	Category string
	Filename string
	Aliases  []string
}

var referenceCatalog = []ReferenceSpec{
	{"Clean Architecture", "Architecture", "architecture-clean.md", nil},
	{"Event-Driven Architecture", "Architecture", "architecture-event-driven.md", nil},
	{"Hexagonal Architecture", "Architecture", "architecture-hexagonal.md", nil},
	{"Layered Architecture", "Architecture", "architecture-layered.md", nil},
	{"Modular Monolith", "Architecture", "architecture-modular-monolith.md", nil},
	{"Ports and Adapters", "Architecture", "architecture-ports-and-adapters.md", nil},
	{"Service-Oriented Architecture", "Architecture", "architecture-service-oriented.md", nil},
	{"Vertical Slice Architecture", "Architecture", "architecture-vertical-slice.md", nil},
	{"Cache-Aside", "Data", "data-cache-aside.md", nil},
	{"Repository", "Data", "data-repository.md", nil},
	{"Unit of Work", "Data", "data-unit-of-work.md", nil},
	{"Dependency Injection", "Dependency", "dependency-injection.md", nil},
	{"Dependency Inversion", "Dependency", "dependency-inversion.md", nil},
	{"Abstract Factory", "GoF", "gof-abstract-factory.md", nil},
	{"Adapter", "GoF", "gof-adapter.md", nil},
	{"Bridge", "GoF", "gof-bridge.md", nil},
	{"Builder", "GoF", "gof-builder.md", nil},
	{"Chain of Responsibility", "GoF", "gof-chain-of-responsibility.md", nil},
	{"Command", "GoF", "gof-command.md", nil},
	{"Composite", "GoF", "gof-composite.md", nil},
	{"Decorator", "GoF", "gof-decorator.md", nil},
	{"Facade", "GoF", "gof-facade.md", nil},
	{"Factory Method", "GoF", "gof-factory-method.md", nil},
	{"Flyweight", "GoF", "gof-flyweight.md", nil},
	{"Interpreter", "GoF", "gof-interpreter.md", nil},
	{"Iterator", "GoF", "gof-iterator.md", nil},
	{"Mediator", "GoF", "gof-mediator.md", nil},
	{"Memento", "GoF", "gof-memento.md", nil},
	{"Observer", "GoF", "gof-observer.md", nil},
	{"Prototype", "GoF", "gof-prototype.md", nil},
	{"Proxy", "GoF", "gof-proxy.md", nil},
	{"Singleton", "GoF", "gof-singleton.md", nil},
	{"State", "GoF", "gof-state.md", nil},
	{"Strategy", "GoF", "gof-strategy.md", nil},
	{"Template Method", "GoF", "gof-template-method.md", nil},
	{"Visitor", "GoF", "gof-visitor.md", nil},
	{"Idempotency", "Integration", "integration-idempotency.md", nil},
	{"Idempotent Consumer", "Integration", "integration-idempotent-consumer.md", nil},
	{"Publish/Subscribe", "Integration", "integration-publish-subscribe.md", nil},
	{"Saga", "Integration", "integration-saga.md", nil},
	{"Transactional Outbox", "Integration", "integration-transactional-outbox.md", nil},
	{"Anti-Corruption Layer", "Modernization", "modernization-anti-corruption-layer.md", nil},
	{"Model-View-Controller", "Presentation", "presentation-model-view-controller.md", []string{"mvc"}},
	{"Circuit Breaker", "Resilience", "resilience-circuit-breaker.md", nil},
	{"Retry and Backoff", "Resilience", "resilience-retry-and-backoff.md", nil},
	{"Timeout and Deadline Propagation", "Resilience", "resilience-timeout-and-deadline.md", nil},
}

func ReferenceCatalog() []ReferenceSpec {
	result := make([]ReferenceSpec, len(referenceCatalog))
	copy(result, referenceCatalog)
	for index := range result {
		result[index].Aliases = append([]string(nil), result[index].Aliases...)
	}
	return result
}

func namedReference(name string) *ReferenceSpec {
	needle := strings.ToLower(strings.TrimSpace(name))
	for index := range referenceCatalog {
		entry := &referenceCatalog[index]
		if strings.ToLower(entry.Name) == needle {
			return entry
		}
		for _, alias := range entry.Aliases {
			if strings.ToLower(alias) == needle {
				return entry
			}
		}
	}
	return nil
}

func explicitReferences(prompt string) []string {
	lower := strings.ToLower(prompt)
	var paths []string
	for _, entry := range referenceCatalog {
		if strings.Contains(lower, strings.ToLower(entry.Name)) {
			paths = append(paths, "references/"+entry.Filename)
			continue
		}
		for _, alias := range entry.Aliases {
			if strings.Contains(lower, strings.ToLower(alias)) {
				paths = append(paths, "references/"+entry.Filename)
				break
			}
		}
	}
	return paths
}

func referenceURL(path string) string {
	return CanonicalReferenceBase + filepath.Base(path)
}
