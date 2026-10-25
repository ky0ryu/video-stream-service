#!/usr/bin/env bash
# verify.sh — status and verification for the kind-based transcoder cluster.
#
# Usage: ./verify.sh [status|smoke|chaos|all]      (default: smoke)
#   status  read-only report: nodes, conditions, capacity, pods, PVCs, restarts, events
#   smoke   non-destructive pass/fail checks (cluster, workloads, dependencies, HTTP, storage)
#   chaos   destructive: deletes pods and checks recovery + Postgres persistence
#   all     smoke, then chaos
#
# Env overrides: NS, CLUSTER, BASE_URL
# Needs: kubectl, docker, curl. Has no path dependencies; run it from anywhere.
set -uo pipefail

NS="${NS:-transcoder}"
CLUSTER="${CLUSTER:-transcoder}"
BASE_URL="${BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0

for tool in kubectl docker curl; do
  command -v "$tool" >/dev/null || { echo "missing required tool: $tool"; exit 2; }
done

section() { echo; echo "== $* =="; }

check() {
  local name="$1"; shift
  local out
  if out=$("$@" 2>&1); then
    echo "PASS  $name"; PASS=$((PASS + 1))
  else
    echo "FAIL  $name"; FAIL=$((FAIL + 1))
    echo "$out" | tail -5 | sed 's/^/        /'
  fi
}

# ---------------------------------------------------------------- status ----
status() {
  section "Nodes"
  kubectl get nodes -o wide

  section "Node conditions (want Ready=True, pressures=False)"
  kubectl get nodes -o custom-columns='NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,MEM_PRESSURE:.status.conditions[?(@.type=="MemoryPressure")].status,DISK_PRESSURE:.status.conditions[?(@.type=="DiskPressure")].status,PID_PRESSURE:.status.conditions[?(@.type=="PIDPressure")].status'

  section "Allocatable per node"
  kubectl get nodes -o custom-columns='NAME:.metadata.name,CPU:.status.allocatable.cpu,MEMORY:.status.allocatable.memory,MAX_PODS:.status.allocatable.pods'

  section "Requests/limits scheduled onto each node"
  kubectl describe nodes | sed -n '/Allocated resources:/,/Events:/p'

  section "kind node containers"
  docker ps --filter "label=io.x-k8s.kind.cluster=$CLUSTER" --format 'table {{.Names}}\t{{.Status}}'

  section "Live usage per kind node (docker stats; metrics-server is not installed)"
  # shellcheck disable=SC2046
  docker stats --no-stream $(docker ps -q --filter "label=io.x-k8s.kind.cluster=$CLUSTER")

  section "Workloads in $NS"
  kubectl -n "$NS" get deploy,pods -o wide

  section "PVCs"
  kubectl -n "$NS" get pvc

  section "Restarts and last termination reason (OOMKilled shows up here)"
  kubectl -n "$NS" get pods -o custom-columns='POD:.metadata.name,RESTARTS:.status.containerStatuses[*].restartCount,LAST_REASON:.status.containerStatuses[*].lastState.terminated.reason'

  section "Ingress controller"
  kubectl -n ingress-nginx get pods

  section "Last 10 warning events in $NS"
  kubectl -n "$NS" get events --field-selector type=Warning --sort-by=.lastTimestamp | tail -10
}

# ----------------------------------------------------------------- tests ----
t_nodes_ready()  { kubectl wait --for=condition=Ready node --all --timeout=30s; }
t_single_node()  { [ "$(kubectl get nodes --no-headers | wc -l)" -eq 1 ] || { echo "multi-node cluster: the RWO video-storage PVC cannot be shared across nodes"; return 1; }; }
t_default_sc()   { kubectl get sc -o jsonpath='{range .items[*]}{.metadata.annotations.storageclass\.kubernetes\.io/is-default-class}{"\n"}{end}' | grep -qx true; }
t_kube_system()  { kubectl -n kube-system wait --for=condition=Ready pod --all --timeout=60s; }
t_ingress_ctrl() { kubectl -n ingress-nginx wait --for=condition=Ready pod -l app.kubernetes.io/component=controller --timeout=60s; }
t_rollout()      { kubectl -n "$NS" rollout status "deploy/$1" --timeout=60s; }
t_pvc_bound()    { [ "$(kubectl -n "$NS" get pvc "$1" -o jsonpath='{.status.phase}')" = Bound ]; }

t_no_restarts() {
  local n
  n=$(kubectl -n "$NS" get pods -o jsonpath='{range .items[*]}{range .status.containerStatuses[*]}{.restartCount}{"\n"}{end}{end}' | awk '{s+=$1} END{print s+0}')
  [ "$n" -eq 0 ] || { echo "total container restarts: $n"; kubectl -n "$NS" get pods; return 1; }
}

t_pg_ready()  { kubectl -n "$NS" exec deploy/postgres -- sh -c 'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"'; }
t_pg_query()  { kubectl -n "$NS" exec deploy/postgres -- sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc "select 1"' | grep -qx 1; }
t_redis()     { kubectl -n "$NS" exec deploy/redis -- redis-cli ping | grep -qx PONG; }

# DNS + TCP from inside the cluster network, same path the app pods use.
t_in_cluster_net() {
  kubectl -n "$NS" run "nettest-$$" --rm -i --restart=Never \
    --image=busybox:1.36 --image-pull-policy=IfNotPresent -- \
    sh -c 'nc -z -w 3 postgres 5432 && nc -z -w 3 redis 6379 && nc -z -w 3 api 8080'
}

t_http() { curl -fsS --max-time 5 "$BASE_URL$1" >/dev/null; }

t_http_eventually() {
  for _ in $(seq 1 30); do
    curl -fsS --max-time 3 "$BASE_URL$1" >/dev/null 2>&1 && return 0
    sleep 2
  done
  echo "$1 never returned 2xx within 60s"; return 1
}

# Write from one pod, read from another, both mounting video-storage.
# This is the exact assumption that breaks on a multi-node cluster.
t_storage_share() {
  local marker="k8s-check-$(date +%s)"
  run_pod() { # name, shell command
    kubectl -n "$NS" run "$1" --rm -i --restart=Never --image=busybox:1.36 \
      --overrides='{"spec":{"containers":[{"name":"'"$1"'","image":"busybox:1.36","imagePullPolicy":"IfNotPresent","command":["sh","-c","'"$2"'"],"volumeMounts":[{"name":"v","mountPath":"/data/videos"}]}],"volumes":[{"name":"v","persistentVolumeClaim":{"claimName":"video-storage"}}]}}'
  }
  run_pod storage-writer "echo $marker > /data/videos/.healthcheck" \
    && run_pod storage-reader "grep -qx $marker /data/videos/.healthcheck && rm /data/videos/.healthcheck"
}

# ----------------------------------------------------------------- smoke ----
smoke() {
  section "Cluster"
  check "all nodes Ready"                    t_nodes_ready
  check "single node (needed by shared PVC)" t_single_node
  check "default StorageClass present"       t_default_sc
  check "kube-system pods Ready"             t_kube_system
  check "ingress-nginx controller Ready"     t_ingress_ctrl

  section "Workloads"
  for d in postgres redis api worker; do check "deployment/$d rolled out" t_rollout "$d"; done
  for p in postgres-data redis-data video-storage; do check "pvc/$p Bound" t_pvc_bound "$p"; done
  check "no container restarts" t_no_restarts

  section "Dependencies"
  check "postgres accepts connections"       t_pg_ready
  check "postgres executes a query"          t_pg_query
  check "redis answers PING"                 t_redis
  check "in-cluster DNS + TCP (pg/redis/api)" t_in_cluster_net

  section "HTTP via ingress ($BASE_URL)"
  check "GET /health/live"                       t_http /health/live
  check "GET /health/ready"                        t_http /health/ready

  section "Shared storage"
  check "write in pod A, read in pod B"      t_storage_share
}

# ----------------------------------------------------------------- chaos ----
wait_ready() { # app label
  for _ in $(seq 1 45); do
    kubectl -n "$NS" get pods -l "app=$1" --field-selector=status.phase=Running \
      -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -q True && return 0
    sleep 2
  done
  echo "app=$1 did not become Ready within 90s"; return 1
}

t_kill_pod() { kubectl -n "$NS" delete pod -l "app=$1" --wait=true && wait_ready "$1"; }

t_pg_persist() {
  local m="chaos-$(date +%s)" rc
  kubectl -n "$NS" exec deploy/postgres -- sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -c \"create table if not exists _chaos(m text); insert into _chaos values ('$m');\"" || return 1
  kubectl -n "$NS" delete pod -l app=postgres --wait=true || return 1
  wait_ready postgres || return 1
  kubectl -n "$NS" exec deploy/postgres -- sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -tAc \"select count(*) from _chaos where m='$m'\"" | grep -qx 1
  rc=$?
  kubectl -n "$NS" exec deploy/postgres -- sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "drop table if exists _chaos"' >/dev/null 2>&1
  return $rc
}

chaos() {
  section "Chaos (destructive)"
  check "api pod killed -> replacement Ready"      t_kill_pod api
  check "api serves /readyz again"                 t_http_eventually /readyz
  check "worker pod killed -> replacement Ready"   t_kill_pod worker
  check "postgres data survives pod deletion"      t_pg_persist
  check "api recovers /readyz after DB restart"    t_http_eventually /readyz
}

# ------------------------------------------------------------------ main ----
case "${1:-smoke}" in
  status) status ;;
  smoke)  smoke ;;
  chaos)  chaos ;;
  all)    smoke; chaos ;;
  *) echo "usage: $0 [status|smoke|chaos|all]"; exit 2 ;;
esac

if [ $((PASS + FAIL)) -gt 0 ]; then
  echo; echo "Passed: $PASS   Failed: $FAIL"
  [ "$FAIL" -eq 0 ]
fi
