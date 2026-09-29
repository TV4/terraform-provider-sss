# ECS paced scale-up acceptance test

`TestAccEcsScalingPacedScaleUp` exercises the real SSS API and requires both
explicit opt-ins. Use only an approved, non-production ECS fixture whose SSS
scaling configuration is dedicated to this test. The fixture must remain
available, and its schedules/capacity effects must be coordinated before the
run. The test creates, replaces, imports, and deletes only the SSS ECS scaling
configuration; it does not create or delete the ECS service itself. Destroy
verifies that the test-owned SSS configuration is gone.

The test submits both zero and nonzero scale-up settings, imports them, resets
each setting independently and together, changes them through the API to test
refresh drift, and checks that a capacity spread of 121 at rate 1 is rejected
without changing the stored configuration.

Required environment inputs:

- `TF_ACC=1` and `SSS_ECS_ACCEPTANCE=1` (both are required; `TF_ACC` alone never
enables this live test).
- `SSS_ECS_ENDPOINT`: SSS API host name without a scheme.
- `SSS_ECS_AUTH_USERNAME` and `SSS_ECS_AUTH_PASSWORD`.
- `SSS_ECS_REGION` and `SSS_ECS_SERVICE_ID`: region and test-owned ECS fixture
  identifier.
- Optional `SSS_ECS_PROTOCOL`, defaulting to `https`.

Run from the repository root:

```sh
export TF_ACC=1 SSS_ECS_ACCEPTANCE=1
export SSS_ECS_ENDPOINT=sss.example.invalid
export SSS_ECS_REGION=eu-west-1
export SSS_ECS_SERVICE_ID=cluster/service
# Obtain SSS_ECS_AUTH_USERNAME and SSS_ECS_AUTH_PASSWORD from the approved secret source.
go test ./internal/provider -run '^TestAccEcsScalingPacedScaleUp$' -count=1 -v
```

Set credentials in the shell or secret manager; do not put credentials in source
control or command logs. The test only calls SSS and does not independently
verify AWS-side fixture health. It requires the SSS deployment containing the
ECS scale-out API fields.
