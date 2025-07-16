<div align="center">

<a href="https://github.com/PythonicVarun/Stratum">
<img src="https://cdn.pythonicvarun.me/stratum/logo.png" alt="Stratum Logo (Redis logo :hehe)" width="200"/>
</a>

# Stratum

**A high-performance, CDN-like caching layer for your databases. 🔥**

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

---

**Stratum** is a highly-configurable, high-performance Go application that acts as a CDN-like layer for your databases. It allows you to expose specific data (like user avatars, configuration JSON, etc.) through custom API endpoints, with a powerful caching layer to protect your databases from excessive load.

## 🚀 Key Features

- **⚡ High Performance**: Built with Go for speed and efficiency.
- **🔄 Dynamic Routes**: Configure custom URL patterns for your data on the fly.
- **🏢 Multi-Project**: Serve data for multiple, independent projects from a single Stratum instance.
- **⚙️ Environment-Driven**: All configuration is managed via environment variables. No code changes needed to add or modify endpoints.
- **📦 Caching**: Built-in support for Redis caching with configurable TTLs per route.
- **⚖️ Scalable**: Designed to be stateless for easy horizontal scaling.

## 🤔 How It Works

The server reads its configuration from environment variables on startup. For each "project" it finds, it dynamically creates an HTTP endpoint. When a request hits an endpoint, Stratum performs the following steps:

```
Request ➡️ [Stratum Endpoint] ➡️ Cache Check ❓
                                    |
                                    ├─ Hit 🎯 ➡️ Serve from Cache ⚡
                                    |
                                    └─ Miss 🤷 ➡️ Query Database 💾 ➡️ Store in Cache 📦 ➡️ Serve Data 🚀
```

## 🏁 Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) (version 1.21+ recommended)
- [Redis](https://redis.io/topics/quickstart)
- A running database instance (e.g., PostgreSQL, MySQL)

### Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/PythonicVarun/Stratum.git
    cd Stratum
    ```

2.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Create a configuration file:**
    Copy the example `.env` file. This file will store your environment variables for local development.
    ```bash
    cp .env.example .env
    ```

## 🛠️ Configuration

Configuration is managed entirely via environment variables, following the principles of a [12-factor app](https://12factor.net/config). For local development, you can set these in your `.env` file.

### Server Configuration

| Variable                | Description                            | Default                    |
|-------------------------|----------------------------------------|----------------------------|
| `SERVER_PORT`           | The port on which the server will run. | `8080`                     |
| `REDIS_URL`             | The connection URL for Redis.          | `redis://localhost:6379/0` |
| `API_CLIENT_USER_AGENT` | The User-Agent header for API sources. | `Pythonic-Stratum-Client`  |

### Project Configuration

To add a new endpoint, you define a set of `PROJECT_n_*` variables, where `n` is a unique number for each project. Each project must have a `PROJECT_n_SOURCE_TYPE`, which can be either `db` or `api`.

#### Source Type: `db`

This is the default source type. It queries a database table to fetch the data.

| Variable                  | Description                                                                    | Example                               |
|---------------------------|--------------------------------------------------------------------------------|---------------------------------------|
| `PROJECT_n_SOURCE_TYPE`   | The source type for the project.                                               | `db`                                  |
| `PROJECT_n_ROUTE`         | The URL pattern. **Must** contain a placeholder in curly braces, like `{id}`.    | `/users/{id}/profile`                 |
| `PROJECT_n_DB_DSN`        | The Data Source Name for the database connection.                              | `user:pass@tcp(127.0.0.1:3306)/db`    |
| `PROJECT_n_TABLE`         | The database table to query.                                                   | `user_profiles`                       |
| `PROJECT_n_ID_COLUMN`     | The column for the `WHERE` clause. **Must** match the placeholder in `ROUTE`.    | `id`                                  |
| `PROJECT_n_SERVE_COLUMN`  | The column whose data should be returned in the response body.                 | `profile_json`                        |
| `PROJECT_n_CONTENT_TYPE`  | The `Content-Type` HTTP header for the response.                               | `application/json`                    |
| `PROJECT_n_CACHE_TTL_SECONDS` | The number of seconds to cache the response. Set to `0` to disable caching. | `3600`                                |

#### Source Type: `api`

This source type fetches data from an external API endpoint.

| Variable                  | Description                                                                    | Example                                               |
|---------------------------|--------------------------------------------------------------------------------|-------------------------------------------------------|
| `PROJECT_n_SOURCE_TYPE`   | The source type for the project.                                               | `api`                                                 |
| `PROJECT_n_ROUTE`         | The URL pattern. **Must** contain a placeholder.                               | `/users/{user_id}/avatar`                             |
| `PROJECT_n_API_ENDPOINT`  | The external API endpoint to call. The placeholder must match `ROUTE`.         | `http://example.com/api/avatars/{user_id}`            |
| `PROJECT_n_ID_COLUMN`     | The name of the placeholder in `ROUTE` and `API_ENDPOINT`.                     | `user_id`                                             |
| `PROJECT_n_CONTENT_TYPE`  | The `Content-Type` HTTP header for the response.                               | `image/png`                                           |
| `PROJECT_n_CACHE_TTL_SECONDS` | The number of seconds to cache the response. Set to `0` to disable caching. | `300`                                                 |

## ▶️ Running the Application

Once your `.env` file is configured, you can run the server:

```bash
go run ./cmd/Stratum
```

The server will start on the port specified by the `SERVER_PORT` environment variable.

## 🤝 Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

Please read our [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## 📜 License

Distributed under the MIT License. See `LICENSE` for more information.

---

<p align="center">Made with ❤️ by Varun Agnihotri</p>
