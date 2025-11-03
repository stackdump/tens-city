---
title: PlantUML Diagrams Guide
description: Create professional UML diagrams with PlantUML's simple text-based syntax
datePublished: 2025-11-03T00:00:00Z
author:
  name: Content Team
  type: Organization
  url: https://tens.city
tags:
  - diagrams
  - plantuml
  - uml
  - visualization
  - tutorial
collection: guides
lang: en
slug: plantuml-diagrams
draft: false
---

# PlantUML Diagrams Guide

Tens City now supports [PlantUML](https://plantuml.com/) diagrams, enabling you to create professional UML diagrams using simple text descriptions.

## Sequence Diagrams

Model interactions between objects over time:

```plantuml
@startuml
actor User
participant "Web Browser" as Browser
participant "API Server" as API
database "Database" as DB

User -> Browser: Open application
Browser -> API: GET /api/data
API -> DB: SELECT * FROM users
DB --> API: Return user data
API --> Browser: JSON response
Browser --> User: Display content
@enduml
```

## Use Case Diagrams

Capture functional requirements:

```plantuml
@startuml
left to right direction
actor "Blog Author" as author
actor "Blog Reader" as reader

rectangle "Tens City Blog System" {
  usecase "Write Post" as UC1
  usecase "Publish Post" as UC2
  usecase "Read Post" as UC3
  usecase "Comment on Post" as UC4
  usecase "Share Post" as UC5
}

author --> UC1
author --> UC2
reader --> UC3
reader --> UC4
reader --> UC5
UC2 .> UC1 : <<include>>
@enduml
```

## Class Diagrams

Model object-oriented structure:

```plantuml
@startuml
class Document {
  -String id
  -String title
  -String content
  -Date publishDate
  +publish()
  +archive()
  +render()
}

class Author {
  -String name
  -String email
  +writePost()
  +editPost()
}

class Comment {
  -String id
  -String text
  -Date created
  +edit()
  +delete()
}

Author "1" --> "*" Document : creates
Document "1" --> "*" Comment : has
Author "1" --> "*" Comment : writes

@enduml
```

## Activity Diagrams

Model workflows and business processes:

```plantuml
@startuml
start
:User visits blog;
if (User authenticated?) then (yes)
  :Show personalized feed;
  :User selects post;
  :Display post content;
  if (User wants to comment?) then (yes)
    :Submit comment;
    :Save comment to database;
  else (no)
    :Continue reading;
  endif
else (no)
  :Show public posts;
  :Prompt for login;
endif
:End session;
stop
@enduml
```

## Component Diagrams

Visualize system architecture:

```plantuml
@startuml
package "Frontend" {
  [Web UI] as UI
  [React Components] as React
}

package "Backend" {
  [API Server] as API
  [Authentication] as Auth
  [Content Service] as Content
}

package "Storage" {
  database "PostgreSQL" as DB
  database "Redis Cache" as Cache
  [Object Storage] as S3
}

UI --> React
React --> API : HTTP/REST
API --> Auth
API --> Content
Auth --> DB
Content --> DB
Content --> Cache
Content --> S3 : Store media
@enduml
```

## State Diagrams

Model object lifecycles:

```plantuml
@startuml
[*] --> Draft : Create post

Draft --> Review : Submit for review
Review --> Draft : Request changes
Review --> Approved : Approve

Approved --> Published : Publish
Published --> Archived : Archive
Archived --> Published : Restore

Published --> [*] : Delete permanently
Archived --> [*] : Delete permanently
@enduml
```

## Deployment Diagrams

Show physical deployment of artifacts:

```plantuml
@startuml
node "User Device" {
  [Web Browser]
}

node "CDN" {
  [Static Assets]
}

node "Web Server" {
  [Nginx]
  [Application]
}

node "Database Server" {
  [PostgreSQL]
}

node "Cache Server" {
  [Redis]
}

[Web Browser] --> [Nginx] : HTTPS
[Web Browser] --> [Static Assets] : HTTPS
[Nginx] --> [Application] : HTTP
[Application] --> [PostgreSQL] : TCP/5432
[Application] --> [Redis] : TCP/6379
@enduml
```

## Timing Diagrams

Show timing constraints and interactions:

```plantuml
@startuml
robust "API Request" as REQ
concise "Rate Limiter" as LIMIT
robust "Database" as DB
concise "Response" as RESP

@0
REQ is Idle
LIMIT is Available
DB is Idle
RESP is Waiting

@100
REQ is Processing
LIMIT is Checking

@150
LIMIT is Allowed
DB is Querying

@300
DB is Returning
RESP is Ready

@400
REQ is Complete
RESP is Sent
@enduml
```

## Object Diagrams

Show specific instances and relationships:

```plantuml
@startuml
object "John's Blog" as blog1 {
  title = "My First Post"
  author = "John Doe"
  published = "2025-11-03"
}

object "Jane's Blog" as blog2 {
  title = "Getting Started"
  author = "Jane Smith"
  published = "2025-11-02"
}

object "Comment #1" as c1 {
  text = "Great post!"
  author = "Reader1"
}

object "Comment #2" as c2 {
  text = "Very helpful"
  author = "Reader2"
}

blog1 --> c1
blog1 --> c2
@enduml
```

## Getting Started

To use PlantUML diagrams in your posts, use fenced code blocks with the `plantuml` language identifier and wrap your diagram with `@startuml` and `@enduml`:

    ```plantuml
    @startuml
    Alice -> Bob: Hello
    Bob --> Alice: Hi there!
    @enduml
    ```

The diagram will be automatically rendered as an SVG image!

## Tips

- Start diagrams with `@startuml` and end with `@enduml`
- Use `->` for solid arrows and `-->` for dashed arrows
- Add colors with `#color` syntax (e.g., `#lightblue`)
- Use skinparam to customize appearance
- Keep diagrams simple and focused for best readability

## Resources

- [PlantUML Official Website](https://plantuml.com/)
- [PlantUML Language Reference](https://plantuml.com/guide)
- [Real World PlantUML](https://real-world-plantuml.com/) - Examples gallery
- [PlantUML Online Editor](https://www.planttext.com/)

Happy modeling! 🎨
