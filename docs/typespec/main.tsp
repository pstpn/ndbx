import "@typespec/http";
import "@typespec/rest";

@service(#{title: "ndbx-service"})
@server("http://localhost:8080", "localhost")

namespace NDBXService;

using TypeSpec.Http;
using TypeSpec.Rest;

model CookieRequest {
    @header("Cookie")
    cookie?: string;
}

@error
model ErrorResponse {
    @header("Set-Cookie")
    setCookie: string;
    message?: string;
}

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

model UserRegisterRequest {
    full_name: string;
    username: string;
    password: string;
}

model UserRegisterResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 201;
}

model LoginRequest {
    username: string;
    password: string;
}

model LoginResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 204;
}

model LogoutResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 204;
}

model LocationInfo {
    address: string;
}

model EventData {
    id: string;
    title: string;
    description?: string;
    location: LocationInfo;
    started_at: string;
    finished_at: string;
    created_at: string;
    created_by: string;
}

model CreateEventRequest {
    title: string;
    address: string;
    description?: string;
    started_at: string;
    finished_at: string;
}

model CreateEventResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 201;
    @body
    body: {id: string}
}

model CreateEventUnauthorizedResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 401;
}

model GetEventsRequest {
    @header("Cookie")
    cookie?: string;
    @query
    title?: string;
    @query
    limit: int64;
    @query
    offset: int64;
}

model GetEventsResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 200;
    events: EventData[];
    count: int64;
}

namespace Api {
    @summary("healthcheck with session")
    @route("/health")
    @get op health(...HealthRequest): HealthResponse;

    @summary("create or extend user's session")
    @route("/session")
    @post op session(...SessionRequest): SessionResponse | ErrorResponse;

    @summary("register a new user")
    @route("/users")
    @post op register(@body body: UserRegisterRequest, ...CookieRequest): UserRegisterResponse | ErrorResponse;

    @summary("authenticate user")
    @route("/auth/login")
    @post op login(@body body: LoginRequest, ...CookieRequest): LoginResponse | ErrorResponse;

    @summary("logout user")
    @route("/auth/logout")
    @post op logout(...CookieRequest): LogoutResponse | ErrorResponse;

    @summary("create a new event")
    @route("/events")
    @post op createEvent(@body body: CreateEventRequest, ...CookieRequest): CreateEventResponse | CreateEventUnauthorizedResponse | ErrorResponse;

    @summary("get events with filtering and pagination")
    @route("/events")
    @get op getEvents(...GetEventsRequest): GetEventsResponse | ErrorResponse;
}
