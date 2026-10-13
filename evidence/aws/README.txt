Generated 2026-10-13T18:56:38Z against the real ALB (prod, us-east-1):
https://haulmer-cards-prod-405309338.us-east-1.elb.amazonaws.com

hey-load.txt                  hey -n 3000 -c 20 on GET /health: 3000 x 200, ~140 req/s
distribution-under-load.txt   300 sequential GET /health sent while hey was running, counted per instance

Two ECS Fargate tasks in two AZs (10.0.0.x and 10.0.1.x) answered, roughly 50/50.
