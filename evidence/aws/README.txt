Generated 2026-10-13T18:56:38Z against the real ALB (prod, us-east-1):
https://haulmer-cards-prod-405309338.us-east-1.elb.amazonaws.com

hey-load.txt                  hey -n 3000 -c 20 on GET /health: 3000 x 200, ~140 req/s
distribution-under-load.txt   300 sequential GET /health sent while hey was running, counted per instance

Two ECS Fargate instances in two AZs (10.0.0.x and 10.0.1.x) answered, roughly 50/50.

Second run with the service scaled to 3 instances (2026-10-13T20:33:10Z), changed through Terraform (desired_count = 3) and applied by the pipeline:

hey-load-3-instances.txt                  hey -n 3000 -c 20 on GET /health: 3000 x 200, ~139 req/s
distribution-under-load-3-instances.txt   300 sequential GET /health sent while hey was running, counted per instance

All three instances answered: 102, 101 and 97 of 300 (about a third each). The three targets were healthy in the target group.

alb-metrics.txt   Application Load Balancer metrics exported from CloudWatch (get-metric-data). RequestCount is exactly 3300 per test
                  (3000 from hey + 300 from the per-instance count), all of them HTTPCode_Target_2XX. This is independent of the app logs,
                  which skip /health on purpose. Timestamps are in UTC-3 (Argentina).

The environment was destroyed after collecting this evidence.
