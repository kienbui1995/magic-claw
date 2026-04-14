# MagiC Cloud — SaaS Managed Platform

**Date:** 2026-04-14
**Status:** Draft
**Author:** MagiC Team

## Overview

MagiC Cloud is the hosted version of MagiC — users sign up, get a managed MagiC instance, and start managing AI workers without any infrastructure setup.

**Value prop:** "From sign-up to first worker in 60 seconds."

## Architecture

```
                    ┌─────────────────────────────────┐
                    │        MagiC Cloud Gateway       │
                    │   (auth, routing, rate limiting)  │
                    └──────────┬──────────────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
        ┌─────┴─────┐   ┌─────┴─────┐   ┌─────┴─────┐
        │ Tenant A   │   │ Tenant B   │   │ Tenant C   │
        │ (isolated) │   │ (isolated) │   │ (isolated) │
        │ PostgreSQL │   │ PostgreSQL │   │ PostgreSQL │
        └───────────┘   └───────────┘   └───────────┘
```

### Tenant Isolation Model

**Shared-process, isolated-database** (Phase 1):
- Single MagiC process handles all tenants
- Each tenant gets a separate PostgreSQL schema
- API key scoped to tenant
- Simpler to operate, sufficient for early scale

**Dedicated instances** (Phase 2, future):
- Each tenant gets a dedicated MagiC container
- Full resource isolation via Kubernetes
- Required for enterprise/compliance customers

## Data Model

```sql
-- Tenants table (control plane DB)
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    owner_email TEXT NOT NULL,
    plan        TEXT NOT NULL DEFAULT 'free',  -- free, pro, enterprise
    api_key     TEXT UNIQUE NOT NULL,
    db_schema   TEXT UNIQUE NOT NULL,           -- PostgreSQL schema name
    status      TEXT NOT NULL DEFAULT 'active', -- active, suspended, deleted
    worker_limit INT NOT NULL DEFAULT 5,
    task_limit   INT NOT NULL DEFAULT 1000,     -- per month
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

-- Usage tracking
CREATE TABLE usage (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id),
    month       DATE NOT NULL,                  -- first day of month
    tasks_used  INT NOT NULL DEFAULT 0,
    workers_registered INT NOT NULL DEFAULT 0,
    api_calls   INT NOT NULL DEFAULT 0,
    UNIQUE(tenant_id, month)
);

-- Billing events
CREATE TABLE billing_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id),
    event_type  TEXT NOT NULL,  -- subscription_created, payment_success, payment_failed
    amount_cents INT,
    stripe_id   TEXT,
    created_at  TIMESTAMPTZ DEFAULT now()
);
```

## Plans & Pricing

| Feature | Free | Pro ($29/mo) | Enterprise |
|---------|------|-------------|------------|
| Workers | 5 | 50 | Unlimited |
| Tasks/month | 1,000 | 50,000 | Unlimited |
| Storage | Memory | PostgreSQL | Dedicated PostgreSQL |
| Webhooks | ✗ | ✓ | ✓ |
| SSE Streaming | ✓ | ✓ | ✓ |
| Custom domain | ✗ | ✓ | ✓ |
| SLA | — | 99.9% | 99.99% |
| Support | Community | Email | Dedicated |

## API Endpoints (Control Plane)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/cloud/signup` | Create tenant account |
| `POST` | `/cloud/login` | Authenticate, get JWT |
| `GET` | `/cloud/me` | Current tenant info |
| `GET` | `/cloud/usage` | Usage stats for current month |
| `POST` | `/cloud/billing/checkout` | Create Stripe checkout session |
| `POST` | `/cloud/billing/webhook` | Stripe webhook handler |
| `PATCH` | `/cloud/settings` | Update tenant settings |

Tenant's MagiC API is accessed via:
```
https://{tenant-slug}.magic.run/api/v1/...
```

## Auth Flow

1. User signs up with email → tenant created with `free` plan
2. API key generated → user uses it for worker registration + API calls
3. JWT issued for dashboard access
4. Stripe Checkout for plan upgrades

## Implementation Phases

### Phase 1 — MVP (this PR)
- [x] Design spec
- [ ] Tenant signup/login API
- [ ] Tenant-scoped request middleware
- [ ] Usage tracking middleware
- [ ] SaaS landing page
- [ ] Basic dashboard (tenant view)

### Phase 2 — Billing
- [ ] Stripe integration (Checkout + webhooks)
- [ ] Plan enforcement (worker/task limits)
- [ ] Usage-based billing for overages

### Phase 3 — Scale
- [ ] Custom domains (Caddy/Cloudflare)
- [ ] Dedicated instances (Kubernetes)
- [ ] SOC 2 compliance
- [ ] Multi-region deployment

## Tech Decisions

- **Auth:** JWT (short-lived) + API keys (long-lived)
- **Billing:** Stripe (Checkout + Customer Portal)
- **Database:** Single PostgreSQL with schema-per-tenant
- **Hosting:** Fly.io or Railway (initial), Kubernetes (scale)
- **Landing page:** Static HTML in `site/cloud/`
