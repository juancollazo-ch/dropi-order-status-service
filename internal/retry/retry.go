package retry

import (
    "context"
    "time"
    "math/rand"
)

func WithRetry(
    ctx context.Context,
    attempts int,
    baseDelay time.Duration,
    fn func() error,
) error {
    var err error

    for i := 1; i <= attempts; i++ {
        // Verificar si el context expiró
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        err = fn()
        if err == nil {
            return nil
        }

        // No hacer sleep en el último intento
        if i == attempts {
            break
        }

        // Backoff exponencial con jitter
        sleep := baseDelay * time.Duration(1<<uint(i-1))
        jitter := time.Duration(rand.Int63n(int64(baseDelay)))
        totalSleep := sleep + jitter

        // Sleep con context awareness
        select {
        case <-time.After(totalSleep):
            // Continuar al siguiente intento
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    return err
}
