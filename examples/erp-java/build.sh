#!/bin/bash
set -e
cd "$(dirname "$0")"

# Build SDK jar if not present
if [ ! -f lib/ggid-sdk-1.0.0.jar ]; then
  echo "Building GGID SDK..."
  cd ../../sdk/java && mvn package -DskipTests -q
  mkdir -p ../../examples/erp-java/lib
  cp target/ggid-sdk-1.0.0.jar ../../examples/erp-java/lib/
  cd ../../examples/erp-java
fi

echo "Building ERP Java Demo..."
mvn package -DskipTests -q

echo "Done. Run with:"
echo "  GGID_URL=https://ggid.iot2.win PORT=8080 java -jar target/erp-java-demo-1.0.0.jar"
