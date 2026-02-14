import "@typespec/http";
import "@typespec/rest";

@service(#{title: "ndbx-service"})
@server("http://localhost:8080", "localhost")

namespace NDBXService;

using TypeSpec.Http;
using TypeSpec.Rest;

@route("/api")
namespace Api {
    @summary("ping")
    @route("/ping")
    @get op ping(): string;
}