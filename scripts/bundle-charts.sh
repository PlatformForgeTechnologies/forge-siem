#!/usr/bin/env bash
# bundle-charts.sh — Package sub-charts into the umbrella's charts/ directory for
# local development and pre-release testing, before charts are published to GHCR OCI.
#
# Usage:
#   ./scripts/bundle-charts.sh [version]
#
# After running, the umbrella chart is self-contained and can be installed with:
#   helm install forge-siem k8s-apps/forge-siem
#
# In production (post v0.2.0 tag), helm dep update will pull from GHCR OCI instead.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
UMBRELLA_CHARTS="$REPO_ROOT/k8s-apps/forge-siem/charts"
VERSION="${1:-0.2.0}"

# Expect forge-siem-platform to be checked out at the same level as forge-siem.
PLATFORM_CHART="$(cd "$REPO_ROOT/../forge-siem-platform/k8s-apps/siem" 2>/dev/null && pwd || true)"

if [ -z "$PLATFORM_CHART" ] || [ ! -d "$PLATFORM_CHART" ]; then
  echo "ERROR: forge-siem-platform not found at ../forge-siem-platform/k8s-apps/siem"
  echo "       Clone forge-siem-platform next to this repo and retry."
  exit 1
fi

echo "Bundling platform chart from $PLATFORM_CHART"

# Stamp the version before packaging.
cp "$PLATFORM_CHART/Chart.yaml" "$PLATFORM_CHART/Chart.yaml.bak"
sed -i.tmp "s/^version:.*/version: $VERSION/" "$PLATFORM_CHART/Chart.yaml"
sed -i.tmp "s/^appVersion:.*/appVersion: \"$VERSION\"/" "$PLATFORM_CHART/Chart.yaml"
rm -f "$PLATFORM_CHART/Chart.yaml.tmp"

helm package "$PLATFORM_CHART" --destination "$UMBRELLA_CHARTS" --version "$VERSION"

# Restore original Chart.yaml.
mv "$PLATFORM_CHART/Chart.yaml.bak" "$PLATFORM_CHART/Chart.yaml"

echo ""
echo "Packaged: $(ls "$UMBRELLA_CHARTS"/siem-*.tgz | tail -1)"
echo ""
echo "To install:"
echo "  helm install forge-siem k8s-apps/forge-siem --namespace siem --create-namespace"
echo ""
echo "To upgrade:"
echo "  helm upgrade forge-siem k8s-apps/forge-siem --namespace siem"
