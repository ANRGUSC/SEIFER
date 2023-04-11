FILE=$1
if [ -f "$FILE" ]; then
  echo "File exists"
  value=$(< "$FILE")
  if [ "$value" = "ready" ]; then
    echo "Equal"
    exit 0
  else
    echo "Not Equal"
    exit 1
  fi
else
  echo "File doesn't exist"
  exit 1
fi