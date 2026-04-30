# NATS MCP Server - Practical Test Plan

This document outlines real-world scenarios to validate the NATS MCP server's effectiveness in assisting developers during debugging and system management.

## Scenario 1: Debugging "No Responders" for a Service
*   **Name:** Service Connectivity & Request-Reply Debugging
*   **Purpose:** Verify that the AI can identify why a request-reply call is failing and help the developer confirm if any service is listening.
*   **Procedure:**
    1.  Attempt to send a request to a non-existent subject `orders.validate` using `nats_request`.
    2.  Observe the error message.
    3.  Use `nats_ping` to confirm the NATS server itself is healthy.
    4.  (Simulate fix) Start a dummy responder or explain that no service is active on that subject.
*   **Expected Result:**
    *   `nats_request` returns a clear error message indicating "no responders".
    *   AI suggests checking if a service is actually subscribed to the subject.

## Scenario 2: End-to-End JetStream Workflow
*   **Name:** JetStream Resource Lifecycle & Observability
*   **Purpose:** Validate the ability to create, monitor, and clean up JetStream resources.
*   **Procedure:**
    1.  Create a new stream named `LOGS` for subject `logs.>` using `jetstream_create_stream`.
    2.  Publish a test message to `logs.error` using `nats_publish`.
    3.  List streams using `jetstream_list_streams` to verify the message count.
    4.  Retrieve the message content using `jetstream_inspect_stream`.
    5.  Purge the stream using `jetstream_purge_stream` and verify it's empty.
    6.  Delete the stream using `jetstream_delete_stream`.
*   **Expected Result:**
    *   Stream is successfully created and visible in the list.
    *   Message counts and data are accurately reported.
    *   Cleanup operations (purge/delete) work without error.

## Scenario 3: Key-Value State Management
*   **Name:** KV Store Configuration & Verification
*   **Purpose:** Ensure the AI can manage application configuration stored in NATS KV.
*   **Procedure:**
    1.  Create a KV bucket named `config` using `kv_create_bucket`.
    2.  Set a configuration key `timeout` to `30s` using `kv_put`.
    3.  Retrieve the key using `kv_get` to verify correctness.
    4.  List buckets using `kv_list_buckets`.
    5.  Delete the key using `kv_delete_key`.
    6.  Delete the bucket using `kv_delete_bucket`.
*   **Expected Result:**
    *   KV operations reflect the expected state changes.
    *   The AI can correctly read back what it wrote.

## Scenario 4: Monitoring Asynchronous Events
*   **Name:** Live Traffic Observation
*   **Purpose:** Use the synchronous subscription tool to "sniff" traffic on a subject.
*   **Procedure:**
    1.  Start `nats_subscribe_sync` on subject `events.>` with a count of 2 and a timeout of 10s.
    2.  In the background, publish two messages to `events.user.login`.
        Example helper: `rtk go run ./scripts/scenario4_publisher.go -initial-delay 5s -delay 500ms -count 2 -subject events.user.login -payload-prefix scenario4`
    3.  Verify that the tool captures and displays both messages.
*   **Expected Result:**
    *   The tool blocks until messages arrive or timeout.
    *   All received messages are formatted and returned to the AI.
*   **Verification Status:** Verified on 2026-05-01 using `events.>` with 2 messages published to `events.user.login`.

## Scenario 5: Request Timeout vs. No Responders
*   **Name:** Slow Service Diagnosis
*   **Purpose:** Distinguish between "no responders" and "responder exists but is too slow" when debugging request-reply flows.
*   **Procedure:**
    1.  Start a test responder on `orders.slow_validate` that intentionally delays its reply longer than the request timeout.
    2.  Send a request to `orders.slow_validate` using `nats_request` with a short timeout such as `500ms`.
    3.  Observe the returned error text.
    4.  Repeat with a longer timeout to confirm the same responder can eventually answer.
*   **Expected Result:**
    *   The short-timeout request returns a timeout-oriented error rather than a "no responders" error.
    *   The AI can explain the difference and recommend checking responder latency, load, or timeout configuration.
*   **Verification Status:** Verified on 2026-05-01. `orders.slow_validate` timed out at `500ms` and returned a valid response at `4000ms` using `scripts/slow_responder.go` with a `2s` delay.

## Scenario 6: JetStream Point-in-Time Inspection
*   **Name:** Message Sequence Lookup
*   **Purpose:** Verify that the AI can inspect a specific JetStream message by sequence number after a stream has accumulated multiple records.
*   **Procedure:**
    1.  Create a stream named `AUDIT` for subject `audit.>`.
    2.  Publish at least three messages to `audit.login`.
    3.  Use `jetstream_list_streams` or `jetstream_inspect_stream` to confirm that multiple messages exist.
    4.  Retrieve sequence `2` using `jetstream_get_message`.
    5.  Delete the stream after verification.
*   **Expected Result:**
    *   The requested sequence returns the exact stored message.
    *   The AI can use sequence-based lookup to isolate one event from a larger stream.
*   **Verification Status:** Verified on 2026-05-01. Stream `AUDIT` stored 3 messages on `audit.login`, and `jetstream_get_message` returned sequence `2` correctly.

## Scenario 7: KV Overwrite and Revision Tracking
*   **Name:** Configuration Change Audit
*   **Purpose:** Ensure the AI can observe state changes when the same KV key is updated multiple times.
*   **Procedure:**
    1.  Create a KV bucket named `settings`.
    2.  Set key `feature_flag` to `off` using `kv_put`.
    3.  Update the same key to `on` using `kv_put`.
    4.  Retrieve the key using `kv_get`.
    5.  Delete the key and bucket after verification.
*   **Expected Result:**
    *   The final read returns `on`.
    *   Revision metadata increases across writes, demonstrating that updates are tracked rather than silently replaced.
*   **Verification Status:** Verified on 2026-05-01. Bucket `settings` returned `feature_flag=on` with revision `2` after two writes.

## Scenario 8: Resource Cleanup Error Handling
*   **Name:** Safe Failure on Missing Resources
*   **Purpose:** Verify that the AI receives actionable errors when attempting to inspect or delete resources that do not exist.
*   **Procedure:**
    1.  Attempt to inspect a non-existent stream such as `MISSING_STREAM` using `jetstream_inspect_stream`.
    2.  Attempt to delete a non-existent stream using `jetstream_delete_stream`.
    3.  Attempt to fetch a key from a non-existent KV bucket using `kv_get`.
    4.  Record the error messages and suggested remediation.
*   **Expected Result:**
    *   Each operation fails cleanly with a useful error message.
    *   The AI can distinguish operator mistakes, missing resources, and connectivity issues without destructive side effects.
*   **Verification Status:** Verified on 2026-05-01. Missing stream operations returned JetStream `404 stream not found`, and missing KV access returned `bucket not found`.

## Scenario 9: Subject Routing Validation
*   **Name:** Stream Binding Accuracy
*   **Purpose:** Confirm that only subjects matching a stream's configured subject pattern are persisted in JetStream.
*   **Procedure:**
    1.  Create a stream named `ORDERS` for subject `orders.>`.
    2.  Publish one message to `orders.created`.
    3.  Publish one message to an unrelated subject such as `payments.captured`.
    4.  List streams and inspect `ORDERS`.
    5.  Delete the stream after verification.
*   **Expected Result:**
    *   Only the `orders.created` message is stored in `ORDERS`.
    *   The AI can explain subject-pattern scope when developers expect a message to appear in the wrong stream.
*   **Verification Status:** Verified on 2026-05-01. Stream `ORDERS` stored only `orders.created`; `payments.captured` was not persisted.

## Scenario 10: Explicit Ack Success Path
*   **Name:** Consumer Ack Confirmation
*   **Purpose:** Verify that a JetStream consumer can explicitly acknowledge a message and prevent redelivery.
*   **Procedure:**
    1.  Create a stream such as `TASKS` for subject `tasks.>`.
    2.  Publish one message to `tasks.created`.
    3.  Create or bind a consumer with explicit ack policy.
    4.  Fetch the message through the consumer.
    5.  Send `ack` for the delivered message.
    6.  Confirm the same message is not redelivered within the ack wait window.
*   **Expected Result:**
    *   The message is delivered once and successfully acknowledged.
    *   No duplicate delivery occurs after `ack`.
*   **Verification Status:** TODO
*   **Blocked By:** Current MCP toolset does not expose consumer creation, fetch, or explicit `ack` operations.

## Scenario 11: Nack and Redelivery Behavior
*   **Name:** Consumer Negative Ack Flow
*   **Purpose:** Verify that a consumer can negatively acknowledge a message and trigger redelivery behavior.
*   **Procedure:**
    1.  Create a stream such as `RETRY` for subject `retry.>`.
    2.  Publish one message to `retry.job`.
    3.  Create or bind a consumer with explicit ack policy and observable redelivery settings.
    4.  Fetch the message through the consumer.
    5.  Send `nack` for the delivered message.
    6.  Confirm the message is redelivered according to consumer policy.
*   **Expected Result:**
    *   The consumer receives the original delivery and a later redelivery after `nack`.
    *   The AI can explain retry behavior and distinguish `nack` from successful processing.
*   **Verification Status:** TODO
*   **Blocked By:** Current MCP toolset does not expose consumer creation, fetch, redelivery inspection, or explicit `nack` operations.

## Scenario 12: Ack Wait Expiry Without Ack
*   **Name:** Unacked Message Recovery
*   **Purpose:** Verify that a message is redelivered when a consumer receives it but neither `ack`s nor `nack`s it before the ack wait expires.
*   **Procedure:**
    1.  Create a stream such as `JOBS` for subject `jobs.>`.
    2.  Publish one message to `jobs.run`.
    3.  Create or bind a consumer with explicit ack policy and a short ack wait.
    4.  Fetch the message without acknowledging it.
    5.  Wait for the ack deadline to expire.
    6.  Confirm the same message is redelivered.
*   **Expected Result:**
    *   The message is redelivered after the ack wait timeout.
    *   The AI can explain that missing acknowledgments and explicit `nack` are different recovery paths.
*   **Verification Status:** TODO
*   **Blocked By:** Current MCP toolset does not expose consumer lifecycle management or acknowledgment-state inspection.

## Execution Summary
Once these scenarios are verified, the NATS MCP server is considered "Production Ready" for local developer integration.
