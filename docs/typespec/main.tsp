import "@typespec/http";
import "@typespec/rest";

@service(#{title: "ndbx-service"})
@server("http://localhost:8080", "localhost")

namespace NDBXService;

using TypeSpec.Http;
using TypeSpec.Rest;

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

@route("/api")
namespace Api {
    @summary("ping")
    @route("/ping")
    @get op ping(): string;

    @summary("create or extend user's session")
    @route("/session")
    @post op session(...SessionRequest): SessionResponse;
}