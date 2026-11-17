#!/usr/bin/env bash
# C3 - Create one ECR repository per service (8 in total).
#
# Every repository gets:
#   * scan on push enabled
#   * a lifecycle rule that keeps only the newest 20 images
#
# The script is idempotent: running it again creates nothing new, changes
# nothing, and exits 0, printing that each repository already exists.
#
# Usage:
#   AWS_REGION=<region> ./scripts/create-ecr-repos.sh
#   (optional) TEAM=visionary-rebel to override the team name
#
# No credentials are read or stored here: the AWS CLI uses whatever
# credentials are already configured in your shell.

set -euo pipefail

TEAM="${TEAM:-visionary-rebel}"
AWS_REGION="${AWS_REGION:-${AWS_DEFAULT_REGION:-}}"
KEEP_IMAGES=20

# Five application services, the two databases and the monitoring image.
SERVICES=(
  web
  menu-api
  orders-api
  kitchen-worker
  stats-api
  postgres
  redis
  ops-monitor
)

if [[ -z "${AWS_REGION}" ]]; then
  echo "ERROR: set AWS_REGION (for example: AWS_REGION=eu-west-1 $0)" >&2
  exit 1
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "ERROR: the AWS CLI (aws) is not installed or not on PATH" >&2
  exit 1
fi

LIFECYCLE_POLICY=$(
  cat <<EOF
{
  "rules": [
    {
      "rulePriority": 1,
      "description": "Keep only the newest ${KEEP_IMAGES} images",
      "selection": {
        "tagStatus": "any",
        "countType": "imageCountMoreThan",
        "countNumber": ${KEEP_IMAGES}
      },
      "action": { "type": "expire" }
    }
  ]
}
EOF
)

created=0
existing=0

for service in "${SERVICES[@]}"; do
  repo="dhaba-${TEAM}/${service}"

  if output=$(aws ecr describe-repositories \
    --region "${AWS_REGION}" \
    --repository-names "${repo}" 2>&1); then
    echo "exists:  ${repo} (already exists)"
    existing=$((existing + 1))
  elif grep -q "RepositoryNotFoundException" <<<"${output}"; then
    aws ecr create-repository \
      --region "${AWS_REGION}" \
      --repository-name "${repo}" \
      --image-scanning-configuration scanOnPush=true \
      --output text \
      --query 'repository.repositoryUri' >/dev/null
    echo "created: ${repo}"
    created=$((created + 1))
  else
    # Credentials, permissions, region or network problem: do not guess.
    echo "ERROR: could not check ${repo}:" >&2
    echo "${output}" >&2
    exit 1
  fi

  # Applying the same settings again is a no-op, and it repairs a repository
  # that someone changed by hand in the console.
  aws ecr put-image-scanning-configuration \
    --region "${AWS_REGION}" \
    --repository-name "${repo}" \
    --image-scanning-configuration scanOnPush=true >/dev/null

  aws ecr put-lifecycle-policy \
    --region "${AWS_REGION}" \
    --repository-name "${repo}" \
    --lifecycle-policy-text "${LIFECYCLE_POLICY}" >/dev/null
done

echo
echo "Done: ${created} created, ${existing} already existed (${#SERVICES[@]} repositories in dhaba-${TEAM}/)."
