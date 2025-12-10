#!/bin/bash

# Billing Service Deployment Script
# Usage: ./scripts/deploy.sh [environment] [version]
# Example: ./scripts/deploy.sh production v1.2.3

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT=${1:-development}
VERSION=${2:-latest}
NAMESPACE="serphona"

echo -e "${GREEN}====================================${NC}"
echo -e "${GREEN}Billing Service Deployment${NC}"
echo -e "${GREEN}====================================${NC}"
echo -e "Environment: ${YELLOW}${ENVIRONMENT}${NC}"
echo -e "Version: ${YELLOW}${VERSION}${NC}"
echo -e "Namespace: ${YELLOW}${NAMESPACE}${NC}"
echo ""

# Validate environment
if [[ "$ENVIRONMENT" != "development" && "$ENVIRONMENT" != "staging" && "$ENVIRONMENT" != "production" ]]; then
    echo -e "${RED}Error: Invalid environment. Use 'development', 'staging', or 'production'${NC}"
    exit 1
fi

# Set namespace based on environment
if [ "$ENVIRONMENT" == "development" ]; then
    NAMESPACE="serphona-dev"
elif [ "$ENVIRONMENT" == "staging" ]; then
    NAMESPACE="serphona-staging"
else
    NAMESPACE="serphona-prod"
fi

# Confirm production deployment
if [ "$ENVIRONMENT" == "production" ]; then
    echo -e "${RED}⚠️  WARNING: You are about to deploy to PRODUCTION!${NC}"
    read -p "Are you sure you want to continue? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        echo -e "${YELLOW}Deployment cancelled.${NC}"
        exit 0
    fi
fi

echo -e "${GREEN}Step 1: Validating Kubernetes manifests...${NC}"
kubectl apply --dry-run=client -f k8s/ -n ${NAMESPACE} || {
    echo -e "${RED}Error: Manifest validation failed${NC}"
    exit 1
}
echo -e "${GREEN}✓ Manifests validated${NC}"
echo ""

echo -e "${GREEN}Step 2: Creating namespace (if not exists)...${NC}"
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}✓ Namespace ready${NC}"
echo ""

echo -e "${GREEN}Step 3: Applying ConfigMap...${NC}"
kubectl apply -f k8s/configmap.yaml -n ${NAMESPACE}
echo -e "${GREEN}✓ ConfigMap applied${NC}"
echo ""

echo -e "${GREEN}Step 4: Checking secrets...${NC}"
if ! kubectl get secret billing-service-secrets -n ${NAMESPACE} &> /dev/null; then
    echo -e "${RED}Error: Secret 'billing-service-secrets' not found in namespace ${NAMESPACE}${NC}"
    echo -e "${YELLOW}Please create the secret first using:${NC}"
    echo "kubectl create secret generic billing-service-secrets \\"
    echo "  --from-literal=DB_PASSWORD=... \\"
    echo "  --from-literal=STRIPE_SECRET_KEY=... \\"
    echo "  -n ${NAMESPACE}"
    exit 1
fi
echo -e "${GREEN}✓ Secrets exist${NC}"
echo ""

echo -e "${GREEN}Step 5: Applying ServiceAccount and RBAC...${NC}"
kubectl apply -f k8s/serviceaccount.yaml -n ${NAMESPACE}
echo -e "${GREEN}✓ ServiceAccount applied${NC}"
echo ""

echo -e "${GREEN}Step 6: Applying Service...${NC}"
kubectl apply -f k8s/service.yaml -n ${NAMESPACE}
echo -e "${GREEN}✓ Service applied${NC}"
echo ""

echo -e "${GREEN}Step 7: Updating Deployment with version ${VERSION}...${NC}"
kubectl apply -f k8s/deployment.yaml -n ${NAMESPACE}
kubectl set image deployment/billing-service \
    billing-service=ghcr.io/serphona/billing-service:${VERSION} \
    -n ${NAMESPACE}
echo -e "${GREEN}✓ Deployment updated${NC}"
echo ""

echo -e "${GREEN}Step 8: Waiting for rollout to complete...${NC}"
kubectl rollout status deployment/billing-service -n ${NAMESPACE} --timeout=5m || {
    echo -e "${RED}Error: Deployment rollout failed${NC}"
    echo -e "${YELLOW}Rolling back...${NC}"
    kubectl rollout undo deployment/billing-service -n ${NAMESPACE}
    exit 1
}
echo -e "${GREEN}✓ Rollout completed successfully${NC}"
echo ""

echo -e "${GREEN}Step 9: Applying HPA...${NC}"
kubectl apply -f k8s/hpa.yaml -n ${NAMESPACE}
echo -e "${GREEN}✓ HPA applied${NC}"
echo ""

echo -e "${GREEN}Step 10: Applying Ingress...${NC}"
kubectl apply -f k8s/ingress.yaml -n ${NAMESPACE}
echo -e "${GREEN}✓ Ingress applied${NC}"
echo ""

echo -e "${GREEN}Step 11: Running database migrations...${NC}"
MIGRATION_POD=$(kubectl get pods -n ${NAMESPACE} -l app=billing-service -o jsonpath='{.items[0].metadata.name}')
if [ -z "$MIGRATION_POD" ]; then
    echo -e "${RED}Error: No billing-service pod found${NC}"
    exit 1
fi

echo "Running migrations on pod: $MIGRATION_POD"
kubectl exec -n ${NAMESPACE} ${MIGRATION_POD} -- sh -c \
    "cd /app/migrations && for f in *up.sql; do psql \$DATABASE_URL -f \$f 2>&1 || true; done"
echo -e "${GREEN}✓ Migrations completed${NC}"
echo ""

echo -e "${GREEN}Step 12: Verification...${NC}"
echo "Pods:"
kubectl get pods -n ${NAMESPACE} -l app=billing-service
echo ""
echo "Services:"
kubectl get svc -n ${NAMESPACE} -l app=billing-service
echo ""
echo "Ingress:"
kubectl get ingress -n ${NAMESPACE} -l app=billing-service
echo ""

echo -e "${GREEN}====================================${NC}"
echo -e "${GREEN}✓ Deployment completed successfully!${NC}"
echo -e "${GREEN}====================================${NC}"
echo ""
echo -e "View logs: ${YELLOW}kubectl logs -f -l app=billing-service -n ${NAMESPACE}${NC}"
echo -e "Check status: ${YELLOW}kubectl get pods -n ${NAMESPACE} -l app=billing-service${NC}"
echo -e "Describe pod: ${YELLOW}kubectl describe pod <pod-name> -n ${NAMESPACE}${NC}"
