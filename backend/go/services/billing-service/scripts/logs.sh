#!/bin/bash

# Billing Service Logs Script
# Usage: ./scripts/logs.sh [environment] [options]

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ENVIRONMENT=${1:-development}
NAMESPACE="serphona"
FOLLOW=${2:--f}

echo -e "${GREEN}====================================${NC}"
echo -e "${GREEN}Billing Service Logs${NC}"
echo -e "${GREEN}====================================${NC}"

# Set namespace based on environment
if [ "$ENVIRONMENT" == "development" ]; then
    NAMESPACE="serphona-dev"
elif [ "$ENVIRONMENT" == "staging" ]; then
    NAMESPACE="serphona-staging"
else
    NAMESPACE="serphona-prod"
fi

echo -e "Environment: ${YELLOW}${ENVIRONMENT}${NC}"
echo -e "Namespace: ${YELLOW}${NAMESPACE}${NC}"
echo ""

# Show available pods
echo -e "${GREEN}Available pods:${NC}"
kubectl get pods -n ${NAMESPACE} -l app=billing-service
echo ""

# Get logs
echo -e "${GREEN}Fetching logs...${NC}"
kubectl logs ${FOLLOW} -l app=billing-service -n ${NAMESPACE} --tail=100 --prefix=true
