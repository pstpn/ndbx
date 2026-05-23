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

model LogoutUnauthorizedResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 401;
}

model LocationInfo {
    address: string;
    city?: string;
}

union EventCategory {
    "meetup",
    "concert",
    "exhibition",
    "party",
    "other"
}

model EventReviews {
    count: int64;
    rating: float64;
}

model EventData {
    id: string;
    title: string;
    category?: EventCategory;
    price?: int64;
    description?: string;
    location: LocationInfo;
    started_at: string;
    finished_at: string;
    created_at: string;
    created_by: string;
    reactions?: EventReactions;
    reviews?: EventReviews;
}

model EventReactions {
    likes: int64;
    dislikes: int64;
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
    id?: string;
    @query
    title?: string;
    @query
    category?: EventCategory;
    @query
    price_from?: int64;
    @query
    price_to?: int64;
    @query
    address?: string;
    @query
    city?: string;
    @query
    date_from?: string;
    @query
    date_to?: string;
    @query
    user_id?: string;
    @query
    user?: string;
    @query
    include?: string;
    @query
    limit?: int64;
    @query
    offset?: int64;
}

model GetEventRequest {
    @path
    id: string;
    @header("Cookie")
    cookie?: string;
    @query
    include?: string;
}

model GetEventResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 200;
    @body
    body: EventData;
}

model PatchEventRequest {
    category?: EventCategory;
    city?: string;
    price?: int64;
}

model PatchEventParams {
    @path
    id: string;
    @header("Cookie")
    cookie?: string;
}

model PatchEventResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 204;
}

model PatchEventUnauthorizedResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 401;
}

model UserData {
    id: string;
    full_name: string;
    username: string;
}

model GetUsersRequest {
    @header("Cookie")
    cookie?: string;
    @query
    id?: string;
    @query
    name?: string;
    @query
    limit?: int64;
    @query
    offset?: int64;
}

model GetUsersResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 200;
    users: UserData[];
    count: int64;
}

model GetUserRequest {
    @path
    id: string;
    @header("Cookie")
    cookie?: string;
}

model GetUserResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 200;
    @body
    body: UserData;
}

model GetUserEventsRequest {
    @path
    id: string;
    @header("Cookie")
    cookie?: string;
    @query
    title?: string;
    @query
    category?: EventCategory;
    @query
    price_from?: int64;
    @query
    price_to?: int64;
    @query
    address?: string;
    @query
    city?: string;
    @query
    date_from?: string;
    @query
    date_to?: string;
    @query
    include?: string;
    @query
    limit?: int64;
    @query
    offset?: int64;
}

model GetEventsResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 200;
    events: EventData[];
    count: int64;
}

model EventReactionParams {
    @path
    id: string;
    @header("Cookie")
    cookie?: string;
}

model EventReactionResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 204;
}

model EventReactionUnauthorizedResponse {
    @statusCode
    statusCode: 401;
}

model ReviewData {
    id: string;
    event_id: string;
    comment: string;
    created_at: string;
    created_by: string;
    rating: int32;
    updated_at: string;
}

model CreateReviewRequest {
    comment?: string;
    rating?: int32;
}

model UpdateReviewRequest {
    comment?: string;
    rating?: int32;
}

model CreateEventReviewParams {
    @path
    event_id: string;
    @header("Cookie")
    cookie?: string;
}

model CreateEventReviewResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 201;
    @body
    body: {id: string}
}

model CreateEventReviewUnauthorizedResponse {
    @statusCode
    statusCode: 401;
}

model CreateEventReviewConflictResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 409;
    message?: string;
}

model GetEventReviewsParams {
    @path
    event_id: string;
    @header("Cookie")
    cookie?: string;
    @query
    limit?: int64;
    @query
    offset?: int64;
}

model GetReviewsResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 200;
    reviews: ReviewData[];
    count: int64;
}

model UpdateEventReviewParams {
    @path
    event_id: string;
    @path
    review_id: string;
    @header("Cookie")
    cookie?: string;
}

model UpdateEventReviewResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 204;
}

model UpdateEventReviewUnauthorizedResponse {
    @statusCode
    statusCode: 401;
}

model UpdateEventReviewNotFoundResponse {
    @header("Set-Cookie")
    setCookie: string;
    @statusCode
    statusCode: 404;
    message?: string;
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
    @post op logout(...CookieRequest): LogoutResponse | LogoutUnauthorizedResponse | ErrorResponse;

    @summary("create a new event")
    @route("/events")
    @post op createEvent(@body body: CreateEventRequest, ...CookieRequest): CreateEventResponse | CreateEventUnauthorizedResponse | ErrorResponse;

    @summary("get events with filtering and pagination")
    @route("/events")
    @get op getEvents(...GetEventsRequest): GetEventsResponse | ErrorResponse;

    @summary("get event by id")
    @route("/events/{id}")
    @get op getEvent(...GetEventRequest): GetEventResponse | ErrorResponse;

    @summary("patch event by id")
    @route("/events/{id}")
    @patch(#{implicitOptionality: true}) op patchEvent(@body body: PatchEventRequest, ...PatchEventParams): PatchEventResponse | PatchEventUnauthorizedResponse | ErrorResponse;

    @summary("like event by id")
    @route("/events/{id}/like")
    @post op likeEvent(...EventReactionParams): EventReactionResponse | EventReactionUnauthorizedResponse | ErrorResponse;

    @summary("dislike event by id")
    @route("/events/{id}/dislike")
    @post op dislikeEvent(...EventReactionParams): EventReactionResponse | EventReactionUnauthorizedResponse | ErrorResponse;

    @summary("create a review for an event")
    @route("/events/{event_id}/reviews")
    @post op createEventReview(@body body: CreateReviewRequest, ...CreateEventReviewParams): CreateEventReviewResponse | CreateEventReviewUnauthorizedResponse | CreateEventReviewConflictResponse | ErrorResponse;

    @summary("get reviews for an event")
    @route("/events/{event_id}/reviews")
    @get op getEventReviews(...GetEventReviewsParams): GetReviewsResponse | ErrorResponse;

    @summary("update a review for an event")
    @route("/events/{event_id}/reviews/{review_id}")
    @patch(#{implicitOptionality: true}) op updateEventReview(@body body: UpdateReviewRequest, ...UpdateEventReviewParams): UpdateEventReviewResponse | UpdateEventReviewUnauthorizedResponse | UpdateEventReviewNotFoundResponse | ErrorResponse;

    @summary("get users")
    @route("/users")
    @get op getUsers(...GetUsersRequest): GetUsersResponse | ErrorResponse;

    @summary("get user by id")
    @route("/users/{id}")
    @get op getUser(...GetUserRequest): GetUserResponse | ErrorResponse;

    @summary("get user events")
    @route("/users/{id}/events")
    @get op getUserEvents(...GetUserEventsRequest): GetEventsResponse | ErrorResponse;
}
