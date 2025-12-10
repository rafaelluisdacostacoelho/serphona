#!/bin/bash

# Billing Service Rollback Script
# Usage: ./scripts/rollback.sh [environment]

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ENVIRONMENT=${1:-development}
NAMESPACE="serphona"

echo -e "${YELLOW}====================================${NC}"
echo -e "${YELLOW}Billing Service Rollback${NC}"
echo -e "${YELLOW}====================================${NC}"
echo -e "Environment: ${YELLOW}${ENVIRONMENT}${NC}"
echo ""

# Set namespace based on environment
if [ "$ENVIRONMENT" == "development" ]; then
    NAMESPACE="serphona-dev"
elif [ "$ENVIRONMENT" == "staging" ]; then
    NAMESPACE="serphona-staging"
else
    NAMESPACE="serphona-prod"
fi

# Confirm production rollback
if [ "$ENVIRONMENT" == "production" ]; then
    echo -e "${RED}⚠️  WARNING: You are about to rollback PRODUCTION!${NC}"
    read -p "Are you sure you want to continue? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        echo -e "${YELLOW}Rollback cancelled.${NC}"
        exit 0
    fi
fi

echo -e "${YELLOW}Current deployment history:${NC}"
kubectl rollout history deployment/billing-service -n ${NAMESPACE}
echo ""

read -p "Enter revision number to rollback to (or press Enter for previous): " REVISION

if [ -z "$REVISION" ]; then
    echo -e "${YELLOW}Rolling back to previous revision...${NC}"
    kubectl rollout undo deployment/billing-service -n ${NAMESPACE}
else
    echo -e "${YELLOW}Rolling back to revision ${REVISION}...${NC}"
    kubectl rollout undo deployment/billing-service --to-revision=${REVISION} -n ${NAMESPACE}
fi

echo -e "${GREEN}Waiting for rollback to complete...${NC}"
kubectl rollout status deployment/billing-service -n ${NAMESPACE} --timeout=5m

echo -e "${GREEN}✓ Rollback completed successfully!${NC}"
echo ""
kubectl get pods -n ${NAMESPACE} -l app=billing-service
