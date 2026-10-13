Generated 2026-10-13T18:56:38Z against the real ALB (prod, us-east-1):
https://haulmer-cards-prod-405309338.us-east-1.elb.amazonaws.com

hey-load.txt                  hey -n 3000 -c 20 on GET /health: 3000 x 200, ~140 req/s
distribution-under-load.txt   300 sequential GET /health sent while hey was running, counted per instance

Two ECS Fargate instances in two AZs (10.0.0.x and 10.0.1.x) answered, roughly 50/50.

Second run with the service scaled to 3 instances (2026-10-13T20:33:10Z), changed through Terraform (desired_count = 3) and applied by the pipeline:

hey-load-3-instances.txt                  hey -n 3000 -c 20 on GET /health: 3000 x 200, ~139 req/s
distribution-under-load-3-instances.txt   300 sequential GET /health sent while hey was running, counted per instance

All three instances answered: 102, 101 and 97 of 300 (about a third each). The three targets were healthy in the target group.
