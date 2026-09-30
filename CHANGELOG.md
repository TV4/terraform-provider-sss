## 1.5.0

FEATURES:

- Add `scale_up_tasks_per_minute` and `scale_up_lead_time_minutes` to `sss_ecs_scaling`
  for paced scale-up and advance scaling. Both default to zero, preserving existing
  behavior.

## 1.4.0

FEATURES:

- Add scheduled scaling resources for ElastiCache Valkey replicas and shards.
- Add scheduled reader scaling resource for Aurora DB clusters.

## 1.2.3

SECURITY UPDATES:

- Update golang.org/x/crypto from 0.43.0 to 0.45.0

## 1.2.2

SECURITY UPDATES:

- Update google.golang.org/grpc from 1.76.0 to 1.79.3
- Update github.com/cloudflare/circl from 1.6.1 to 1.6.3

## 1.2.0

FEATURES:
Add support for scaling DynamoDB provisioned capacity

## 1.1.0

FEATURES:
Add support for multi-region SSS.

## 1.0.0

FEATURES:
Initial implementation which supports ECS scaling.
