# Ballerina Nutcracker Governance

This document describes the governance model for the Ballerina Nutcracker project. Ballerina Nutcracker follows CNCF-style project governance principles and the CNCF Code of Conduct.

## Mission

Ballerina Nutcracker is a native Ballerina interpreter written in Go: it compiles Ballerina source to Ballerina Intermediate Representation (BIR) and interprets the BIR directly, with a focus on speed, low memory use, and fast startup.

The project aims to:

- Provide a fast, lightweight, native runtime for the Ballerina language, suited to cloud-native and short-lived workloads
- Grow language and standard library support incrementally, organized by well-defined subsets
- Encourage open collaboration and community-driven development
- Foster a welcoming, inclusive, and sustainable contributor community

## CNCF Alignment

Ballerina Nutcracker is not currently a CNCF-hosted project, but is developed in alignment with CNCF community norms, including:

- The CNCF Code of Conduct
- CNCF-style, consensus-driven governance

Ballerina Nutcracker was initially created by contributors from WSO2. Governance decisions are made collectively by the maintainer group and are not controlled by any single organization.

## Roles and Responsibilities

### Maintainers

Maintainers have write access to the Ballerina Nutcracker GitHub repository and are responsible for the overall health, direction, and sustainability of the project. The current list of maintainers is kept in [MAINTAINERS.md](./MAINTAINERS.md), organized into two tiers:

- **Core Maintainers** have merge authority across the entire repository, own the project's long-term roadmap and architectural consistency, and collectively serve as the project's Security Team.
- **Module Maintainers** have merge privileges within specific functional areas of the codebase and participate in cross-project decisions that affect those areas.

Maintainer responsibilities include:

- Defining and evolving the project roadmap and architecture
- Reviewing and approving code and documentation changes
- Ensuring consistency with architectural and design principles
- Managing releases and versioning
- Upholding the CNCF Code of Conduct
- Supporting and mentoring contributors

Being a maintainer is a privilege that comes with responsibility. Maintainers are expected to act in the best interests of the project and its community.

### Contributors

Contributors are anyone who participates in the project through code contributions, documentation, issue reporting, reviews, design discussions, or community support.

All contributors are encouraged to:

- Submit issues and pull requests
- Participate in technical and design discussions
- Review code and documentation
- Help other community members

No formal approval is required to become a contributor.

## Decision Making

Ballerina Nutcracker follows a consensus-driven decision-making model inspired by other CNCF projects.

### Lazy Consensus

Most day-to-day decisions are made using lazy consensus. Proposals are considered accepted if no substantive objections are raised within a reasonable time period (typically 3–5 business days).

### Voting

Formal votes are used when consensus cannot be reached or when required for specific governance decisions.

Voting rules:

- One vote per maintainer
- Votes may be conducted asynchronously via GitHub issues, pull requests, GitHub Discussions, or the project's Discord

A formal vote is required for:

- Adding new maintainers
- Removing maintainers
- Major architectural or governance changes

## Working Groups

Working Groups (WGs) are time-bounded groups formed to address a specific architectural or cross-cutting concern that benefits from focused community effort — for example, designing a new language subset, evolving the BIR format, or scoping a new standard library module. A working group produces concrete deliverables, typically architecture specifications, design documents, or implementation guidelines, and dissolves once those deliverables are accepted.

### Forming a Working Group

Any maintainer may propose a working group by opening a pull request that adds a `CHARTER.md` to `working-groups/<wg-name>/`. The charter must define the WG's objective, scope, deliverables, timeline, and proposed lead. At least one maintainer approval is required to merge.

### Working Group Lead, Co-Lead(s), Core Members and Community Contributors

Each WG must have a lead who is a current maintainer or active contributor with demonstrated familiarity with the problem domain. The lead chairs meetings, drives the agenda, owns deliverables, and reports progress to the maintainer group. Co-Lead(s) support the WG Lead and cover in their absence, appointed by the WG Lead with agreement from core members. Core Members are maintainers and active contributors who commit to attending meetings and contributing to at least one deliverable. Community Contributors can be anyone from the wider Ballerina Nutcracker community who participates in discussions or provides feedback. No formal commitment required.

A WG should have at least three members (the lead plus two Co-Leads/Core Members) so it can always produce the distinct WG-member approvals required to merge its outputs (see Outputs below). Smaller WGs fall back to the rule described there.

### Working Group Decision Making

Working groups use lazy consensus (3 business days) for day-to-day decisions. Decisions that modify a public API contract or cross-stage interface — for example, the BIR format, a `lib/langlibs`/`lib/stdlibs` module contract, or the Platform Adaptation Layer (`platform/pal`) — also require approval from at least one maintainer outside the WG. That member cannot be part of the WG, such as the WG lead, co-lead(s), or core members.

### Outputs

WG outputs (architecture docs, design specs, RFCs) are merged to the repository via standard pull requests requiring two approvals from WG members and one maintainer approval, all three from distinct individuals other than the PR author. If the WG doesn't have two other eligible members to provide those approvals, an outside maintainer's approval substitutes for each missing WG-member approval. Outputs do not constitute implementation authority — implementation work follows the standard contribution process.

### Dissolving a Working Group

A WG closes when all charter deliverables are accepted or when the WG lead declares the effort complete. The lead notifies the maintainer group via a PR that archives the WG directory and updates its `CHARTER.md` status to `Completed`.

### Active Working Groups

| Working Group | Lead | Status | Charter |
|---|---|---|---|

## Becoming a Maintainer

New maintainers are selected based on demonstrated merit and sustained contributions to the project. Typical criteria include:

- Sustained and meaningful contributions over time (code, documentation, design discussions, or reviews)
  - perform reviews for at least 10 non-trivial PRs
  - contribute at least 20 non-trivial PRs and have them merged
- Demonstrated technical leadership and architectural understanding
- Active participation in project discussions and reviews
- Familiarity with project workflows, contribution guidelines, and quality standards
- A track record of constructive collaboration within the community

A new maintainer must be nominated by an existing maintainer and approved by a **supermajority (at least two-thirds)** of the current maintainers.

## Removing a Maintainer

Maintainers may step down voluntarily at any time.

A maintainer may be removed due to:

- Prolonged inactivity (typically six months or more without meaningful participation, unless a planned return is communicated)
- Repeated failure to meet maintainer responsibilities
- Violation of the CNCF Code of Conduct
- Actions detrimental to the project or community

Removal requires approval by a **supermajority** of the remaining maintainers.

## Meetings

Where feasible, maintainers are encouraged to participate in public community meetings, office hours, or maintainer syncs.

Closed meetings may be held to address security issues or Code of Conduct matters. All maintainers, except those directly involved in a reported Code of Conduct issue, must be invited to such meetings.

## Code of Conduct

Ballerina Nutcracker adheres to the CNCF Code of Conduct. All participants are expected to follow it in all project spaces. See [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md).

## Security

Security vulnerabilities should be reported privately following the process described in [SECURITY.md](./SECURITY.md).

## Amendments

Changes to this governance document require:

- A pull request proposing the change
- Approval by a **supermajority** of maintainers

## Acknowledgements

This governance model is inspired by governance documents from CNCF Sandbox and Incubating projects and is intentionally lightweight to match the project's current size, with room to add more structure as the community grows.
