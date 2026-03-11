import "@typespec/http";
import "@typespec/rest";

@service(#{title: "ndbx-service"})
@server("http://localhost:8080", "localhost")

namespace NDBXService;

using TypeSpec.Http;
using TypeSpec.Rest;

model HealthRequest {
    @header("Cookie")
    cookie?: string;
}

model HealthResponse {
    @header("Set-Cookie")
    setCookie?: string;
    status: string;
}

model SessionRequest {
    @header("Cookie")
    cookie?: string;
}

model SessionResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 200 | 201;
}

namespace Api {
    @summary("healthcheck with session")
    @route("/health")
    @get op health(...HealthRequest): HealthResponse;

    @summary("create or extend user's session")
    @route("/session")
    @post op session(...SessionRequest): SessionResponse;
}