# Veritect

## Product Classification

Veritect is a stateless, zero-trust schema drift detection utility for automated deployment workflows. It is engineered to ensure the integrity and stability of database schemas in continuous integration environments.

## Architecture & Security Model

Veritect operates under a strict zero-trust posture. It processes only structural database metadata (information_schema) natively within the runner environment, ensuring absolute separation from application data records. Database credentials remain securely within the execution environment and are never transmitted externally.

## The Deterministic Core

Veritect employs an alphabetical sorting constraint to guarantee zero false-positive build failures. This deterministic approach ensures consistent and reliable detection of schema drift, maintaining operational continuity.

## Continuous Integration Specification

Below is an example of integrating Veritect within a GitHub Actions pipeline:

```yaml
- name: Check Schema Drift
  run: go run ./cmd/veritect
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
    SLACK_WEBHOOK: ${{ secrets.SLACK_WEBHOOK }}
```

This configuration allows Veritect to execute seamlessly within CI workflows, providing immediate feedback on schema integrity.

## Licensing and Compliance

Veritect is licensed under the Business Source License (BSL) 1.1. This framework allows free usage for internal operations while protecting commercial boundaries. The license will transition to Apache-2.0 on 2030-05-31. Internal and non-commercial use is unrestricted, whereas competing commercial SaaS offerings require a separate license.
