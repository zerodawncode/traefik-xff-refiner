<p align="center"><img src="https://github.com/zerodawncode/traefik-xff-refiner/blob/main/.assets/icon.svg?raw=true" alt="logo" height="150" width="150"></p>

# 🛡️ traefik-xff-refiner

**traefik-xff-refiner** is a high-performance [Traefik](https://traefik.io) plugin designed to intelligently refine the `X-Forwarded-For` header. It allows you to select a specific IP address from the request chain and ensure it is the only one passed to your backend services.

## ✨ Features

- 🎯 **Precise IP Extraction**: Select an IP by its position (depth) in the `X-Forwarded-For` chain.
- 🔄 **Negative Depth Support**: Easily pick the last hop or immediate peer using negative indices (e.g., `-1`).
- 🧹 **Header Cleanup**: Automatically overwrites `X-Forwarded-For` with the selected IP, removing proxy clutter.
- 🕵️‍♂️ **Auditability**: Preserves the full original chain in `X-Original-Forwarded-For`.
- 🛡️ **RemoteAddr Override**: Optionally force Traefik's internal `RemoteAddr` to the selected IP for seamless backend integration.
- 🚀 **Zero Dependencies**: Lightweight and optimized for Traefik's Go-based plugin architecture.

## ⚙️ How It Works

For every incoming request, the plugin:
1. Parses the `X-Forwarded-For` header and includes the direct `RemoteAddr`.
2. Extracts the IP based on your configured `depth` (defaults to `0`, the first/client IP).
3. Validates that the selected entry is a real IPv4/IPv6 address. If it is not (for
   example a forged `X-Forwarded-For: localhost`), or if `depth` points outside the
   chain, it falls back to the immediate peer (`RemoteAddr`), the one hop a client
   cannot forge.
4. Updates `X-Forwarded-For`, `X-Real-Ip`, and `X-Forwarded-For-Proxy-Protocol` to this selected IP.
5. If `overrideRemoteAddr` is `true` (default), it clears the `X-Forwarded-For` header so Traefik appends *only* your selected IP.

## 🚀 Installation & Usage

### 1. Static Configuration

Add the plugin to your Traefik static configuration:

```yaml
experimental:
  plugins:
    traefik-xff-refiner:
      moduleName: github.com/zerodawncode/traefik-xff-refiner
      version: v1.0.2
```

### 2. Dynamic Middleware Configuration

Apply the middleware to your routers:

```yaml
http:
  middlewares:
    xff-refiner:
      plugin:
        traefik-xff-refiner:
          depth: -2             # see "Choosing depth" below
          overrideRemoteAddr: true # Ensure backend sees exactly one IP in XFF
```

### 3. Choosing `depth`

`X-Forwarded-For` is client supplied: anything to the left of what your trusted
proxies append is under the attacker's control. Count from the right, never the left:

| Topology | `depth` | Why |
|---|---|---|
| Traefik is the first proxy the client reaches | `-1` | `RemoteAddr` is the client |
| Cloudflare (or any proxy that *appends* the visitor IP) in front of Traefik | `-2` | the hop before the edge is the visitor |

`depth: 0` (the default, kept for backward compatibility) selects the leftmost entry and
should only be used when Traefik's `forwardedHeaders.trustedIPs` already guarantees the
header was rewritten by a trusted hop. Since v1.0.2 a non-IP entry at the selected depth
can no longer reach the backend, but a *valid-looking* forged IP still can.

## 📝 License

This project is licensed under the MIT License.

- Developed and maintained © 2026 Zerodawn