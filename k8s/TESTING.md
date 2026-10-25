# kind cluster test plan

Scope: verify the local kind cluster and the transcoder deployment on it.
Not in scope: transcoding correctness, load testing, production readiness.

Run order: `./verify.sh status` (look), `./verify.sh smoke` (automated checks), manual tests M1–M6, then `./verify.sh chaos`.

## Preconditions

- `docker`, `kind`, `kubectl`, `curl` installed
- `./setup.sh` completed; cluster `transcoder` exists
- API exposes `/healthz` and `/readyz`
- Ingress reachable at `http://localhost:8080` (override with `BASE_URL`)

## Automated checks (`./verify.sh smoke`)

| ID | Check | Failure usually means |
|----|-------|-----------------------|
| C1 | All nodes Ready | kind node container stopped or Docker out of resources |
| C2 | Exactly one node | Extra node added; RWO `video-storage` PVC is no longer safely shareable |
| C3 | Default StorageClass exists | PVCs stay Pending forever |
| C4 | kube-system pods Ready | Cluster itself unhealthy (DNS/CNI); nothing else is trustworthy |
| C5 | ingress-nginx controller Ready | Image pull still in progress, or node missing the `ingress-ready=true` label |
| W1 | postgres, redis, api, worker rolled out | Check `kubectl -n transcoder describe pod` and `logs --previous` |
| W2 | postgres-data, redis-data, video-storage Bound | Pod never scheduled (`WaitForFirstConsumer`), or no default StorageClass |
| W3 | Zero container restarts | Probe failing, OOMKill, or crash loop; see the restart reason in `status` |
| D1 | Postgres accepts connections and runs a query | Secret/env mismatch, or data dir init failed |
| D2 | Redis answers PING | Redis pod not ready |
| D3 | In-cluster DNS + TCP to postgres, redis, api | Service selector or label mismatch; CoreDNS down |
| H1 | `GET /healthz`, `GET /readyz` via ingress | Ingress rule, port mapping, or API probe endpoints wrong |
| S1 | Write in pod A, read in pod B on `video-storage` | PVC not shareable (multi-node) or wrong mount path |

## Manual checks

These need application knowledge the script doesn't have. Fill in the routes for your API.

| ID | Test | Steps | Expected |
|----|------|-------|----------|
| M1 | Upload through ingress | Upload a video larger than 1 MB (nginx default limit) to the API via `localhost:8080` | Accepted; no 413. Confirms `proxy-body-size: "0"` works |
| M2 | End-to-end transcode | Upload, wait for the Asynq job, fetch the output | Output file exists on `video-storage`; DB row shows completed |
| M3 | Worker killed mid-job | Start a long transcode, `kubectl -n transcoder delete pod -l app=worker` | Job is retried/requeued and completes; no corrupt partial output left behind |
| M4 | Graceful shutdown | Start a transcode, run `kubectl -n transcoder rollout restart deploy/worker` | In-flight job finishes (or requeues cleanly) within the 120s grace period; compare with and without SIGTERM handling |
| M5 | OOMKill behavior | `kubectl -n transcoder set resources deploy/worker --limits=memory=16Mi`, watch pods, then `kubectl apply -f manifests/` to restore | `lastState.terminated.reason: OOMKilled`, then `CrashLoopBackOff`; you can explain each state |
| M6 | Idempotent setup | Run `./setup.sh` a second time | No errors; existing cluster reused; workloads still healthy |

## Known data-loss behavior (verify once, so it never surprises you)

The default kind StorageClass uses reclaim policy `Delete`. Deleting a PVC or the `transcoder` namespace destroys Postgres data and stored videos. Deleting only a *pod* does not. `./verify.sh chaos` checks the pod case.

## Debug cheat sheet

```
kubectl -n transcoder get pods -o wide
kubectl -n transcoder describe pod <pod>          # events, probe failures, scheduling
kubectl -n transcoder logs <pod> --previous       # logs from the crashed container
kubectl -n transcoder get events --sort-by=.lastTimestamp
kubectl -n ingress-nginx logs -l app.kubernetes.io/component=controller
docker exec -it transcoder-control-plane crictl ps   # what the node is actually running
```

## Pass criteria

All automated checks pass, M1–M6 behave as expected, and `./verify.sh chaos` passes. Record any deviation here before moving on to HPA/KEDA work.
