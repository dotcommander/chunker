# Chunker Documentation

Complete documentation for the Chunker text chunking service.

## Quick Links

- [Installation & Quick Start](#getting-started)
- [Choose a Chunking Strategy](#strategies)
- [API Endpoints](#api-reference)
- [Add a New Strategy](#contributing)

## Table of Contents

- [Getting Started](#getting-started)
- [Concepts](#concepts)
- [Strategies](#strategies)
- [API Reference](#api-reference)
- [Contributing](#contributing)

---

## Getting Started

### [Installation](../README.md#installation)

Install dependencies and build the binary.

### [Quick Start](../README.md#quick-start)

Get up and running with server or CLI mode in minutes.

### [Basic Usage](../README.md#usage)

Simple examples for common chunking tasks.

---

## Concepts

### [Architecture Overview](ARCHITECTURE.md)

Clean architecture principles with four-layer separation (Domain, Service, Chunking, Handler). Learn about the project structure, design patterns, and component relationships.

### [Knowledge Base](KNOWLEDGE.md)

Advanced concepts, design decisions, and implementation insights.

---

## Strategies

### [Smart Boundary](strategies/smart-boundary.md)

Abbreviation-aware, punctuation-based sentence detection delegated to [github.com/dotcommander/reliquary](https://github.com/dotcommander/reliquary). Handles abbreviations and decimal numbers. **Recommended for most natural language text.**

### [Sentence Boundary](strategies/sentence-boundary.md)

Simple punctuation-based sentence splitting. Fast and lightweight alternative to smart_boundary with limited intelligence.

### [Word Boundary](strategies/word-boundary.md)

Splits at word boundaries, never breaking words mid-word. Ideal for search indexing and content display scenarios.

---

## API Reference

### [CLI Reference](api/cli.md)

Command-line interface documentation. Covers CLI mode, server mode, flags, and usage examples.

### [API Schemas](api/schemas.md)

Request and response schemas for the `/chunk` endpoint. Includes validation rules and all supported parameters.

### [Error Handling](api/errors.md)

Comprehensive error reference with HTTP status codes, validation errors, and error response structures.

### [Legacy API](API.md)

Complete API documentation including endpoints, authentication, and integration examples.

---

## Contributing

### [Architecture Overview](contributing/architecture.md)

Deep dive into the domain, service, handler, and CLI layers plus Reliquary delegation. Essential reading for contributors.

### [Adding a New Strategy](contributing/new-strategy.md)

Guide to adding a strategy in Reliquary, exposing its matching domain constant here, and testing the integration.

### [Testing Guidelines](contributing/testing.md)

Testing best practices, table-driven tests, and coverage requirements. (Coming soon)
