---
title: Mermaid Diagrams Guide
description: Learn how to create beautiful flowcharts, sequence diagrams, and more with Mermaid
datePublished: 2025-11-03T00:00:00Z
author:
  name: Content Team
  type: Organization
  url: https://tens.city
tags:
  - diagrams
  - mermaid
  - visualization
  - tutorial
collection: guides
lang: en
slug: mermaid-diagrams
draft: false
---

# Mermaid Diagrams Guide

Tens City now supports [Mermaid](https://mermaid.js.org/) diagrams, allowing you to create beautiful visualizations using simple text syntax.

## Flowcharts

Create flowcharts to visualize processes and decision trees:

```mermaid
graph TD;
    A[Start] --> B{Is it working?};
    B -->|Yes| C[Great!];
    B -->|No| D[Debug];
    D --> E[Fix the issue];
    E --> B;
    C --> F[End];
```

## Sequence Diagrams

Illustrate interactions between different actors or systems:

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant Server
    participant Database
    
    User->>Browser: Enter URL
    Browser->>Server: HTTP Request
    Server->>Database: Query data
    Database-->>Server: Return results
    Server-->>Browser: HTTP Response
    Browser-->>User: Display page
```

## Class Diagrams

Document object-oriented relationships:

```mermaid
classDiagram
    class Animal {
        +String name
        +int age
        +makeSound()
    }
    class Dog {
        +String breed
        +bark()
    }
    class Cat {
        +String color
        +meow()
    }
    Animal <|-- Dog
    Animal <|-- Cat
```

## State Diagrams

Model state machines and transitions:

```mermaid
stateDiagram-v2
    [*] --> Draft
    Draft --> Review: Submit
    Review --> Draft: Request changes
    Review --> Approved: Accept
    Approved --> Published: Publish
    Published --> Archived: Archive
    Archived --> [*]
```

## Gantt Charts

Plan and track project timelines:

```mermaid
gantt
    title Project Timeline
    dateFormat  YYYY-MM-DD
    section Planning
    Requirements gathering    :a1, 2025-11-01, 5d
    Design specifications     :a2, after a1, 7d
    section Development
    Backend implementation    :b1, after a2, 14d
    Frontend implementation   :b2, after a2, 14d
    section Testing
    Integration testing       :c1, after b1, 5d
    User acceptance testing   :c2, after c1, 3d
```

## Pie Charts

Visualize proportions and percentages:

```mermaid
pie title Programming Languages Used
    "Go" : 45
    "JavaScript" : 25
    "Python" : 15
    "TypeScript" : 10
    "Other" : 5
```

## Entity Relationship Diagrams

Model database relationships:

```mermaid
erDiagram
    USER ||--o{ POST : creates
    USER ||--o{ COMMENT : writes
    POST ||--o{ COMMENT : has
    USER {
        string id
        string username
        string email
    }
    POST {
        string id
        string title
        string content
        date published
    }
    COMMENT {
        string id
        string text
        date created
    }
```

## Git Graphs

Visualize branching and merging:

```mermaid
gitGraph
    commit
    commit
    branch develop
    checkout develop
    commit
    commit
    checkout main
    merge develop
    commit
    branch feature
    checkout feature
    commit
    checkout develop
    merge feature
    checkout main
    merge develop
```

## Getting Started

To use Mermaid diagrams in your posts, simply use fenced code blocks with the `mermaid` language identifier:

    ```mermaid
    graph LR;
        A-->B;
    ```

The diagram will be automatically rendered when your content is displayed!

## Resources

- [Mermaid Official Documentation](https://mermaid.js.org/)
- [Mermaid Live Editor](https://mermaid.live/) - Test your diagrams
- [Mermaid Cheat Sheet](https://jojozhuang.github.io/tutorial/mermaid-cheat-sheet/)

Happy diagramming! 📊
