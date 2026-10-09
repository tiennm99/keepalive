# Adding an adapter

An adapter connects to one datastore, increments a counter on every tick, and releases its resources on close. Adapters live in the `adapter` package, one file each.

1. **Create `adapter/<name>.go`** and implement the `Adapter` interface from [`adapter/adapter.go`](../adapter/adapter.go):

   ```go
   type Adapter interface {
   	Connect(ctx context.Context) error           // open the client and create the counter if missing
   	Increment(ctx context.Context) (int64, error) // add 1 and return the new value
   	Close(ctx context.Context) error              // release resources; called after every session
   }
   ```

2. **Register the factory and its config keys** in `init()`. The factory only reads config and must not do network I/O, because keepalive calls it at startup to validate the file.

   ```go
   func init() {
   	ConfigKeys["<name>"] = []string{"url"}
   	Registry["<name>"] = func(cfg Config) (Adapter, error) {
   		url, err := cfg.Required("url")
   		if err != nil {
   			return nil, err
   		}
   		return &myAdapter{url: url, key: cfg.Optional("counter_key", "counter")}, nil
   	}
   }
   ```

   `ConfigKeys` drives the unknown-key warnings; a test fails if an adapter is registered without it. `counter_key` is supplied by the config loader and is not listed.

3. **Follow the conventions** that keep the runner's guarantees:
   - Honour `ctx` in `Connect` and `Increment`; the runner bounds both.
   - Store the client on the adapter only after `Connect` succeeds, so `Close` never closes a failed client twice.
   - Make `Connect` idempotent: it re-runs after every failure and should recreate a missing counter.
   - Implement `ConnectTimeout() time.Duration` only if setup legitimately needs longer than 1 minute.
   - Keep the driver pure Go (`CGO_ENABLED=0`).

4. **Document it:** add the adapter to [`config.example.yml`](../config.example.yml), its keys to [Configuration](configuration.md), its writes to [How it works](how-it-works.md), and a row to the README's adapter table.

5. **Test it:** `go vet ./...` and `go test -race ./...`.
