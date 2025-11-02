# GitHub Copilot Custom Agents

This directory contains custom agent configurations for GitHub Copilot. These agents provide specialized assistance for different aspects of the tens-city project.

## Available Agents

### go-expert
**Specialization**: Go language best practices and tens-city codebase patterns

Use this agent for:
- Writing idiomatic Go code
- Following project-specific patterns
- Error handling and security
- Code refactoring and optimization
- Understanding the codebase architecture

### jsonld-specialist
**Specialization**: JSON-LD, semantic web, and schema.org

Use this agent for:
- Creating and validating JSON-LD documents
- Working with schema.org vocabulary
- Understanding URDNA2015 canonicalization
- Content-addressable storage (CID) concepts
- Markdown to JSON-LD mapping

### test-specialist
**Specialization**: Testing patterns and coverage

Use this agent for:
- Writing comprehensive test suites
- Table-driven test patterns
- HTTP handler testing
- Integration and unit tests
- Improving test coverage

### documentation-expert
**Specialization**: Documentation and technical writing

Use this agent for:
- Updating README and documentation files
- API documentation
- Code comments and examples
- User guides and tutorials
- Maintaining documentation quality

## How to Use

GitHub Copilot can automatically detect and use these agents when working on related tasks. You can also explicitly invoke them through GitHub Copilot's interface when available.

For more general project context, see the [copilot-instructions.md](../copilot-instructions.md) file.

## Agent Configuration Format

Each agent is defined in a Markdown file with YAML frontmatter:

```yaml
---
name: agent-name
description: Brief description of the agent's specialization
---

Detailed instructions and context for the agent...
```

## Updating Agents

When updating agent configurations:
1. Ensure YAML frontmatter is valid
2. Keep instructions clear and actionable
3. Update based on project evolution
4. Test changes by using the agent on real tasks
5. Document any new patterns or requirements

## Learn More

- [GitHub Copilot Custom Agents Documentation](https://docs.github.com/en/copilot/concepts/agents/coding-agent/about-custom-agents)
- [Creating Custom Agents Guide](https://docs.github.com/en/copilot/tutorials/customization-library/custom-agents/your-first-custom-agent)
