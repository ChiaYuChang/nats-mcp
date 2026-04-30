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
    2.  In the background (manually or via script), publish two messages to `events.user.login`.
    3.  Verify that the tool captures and displays both messages.
*   **Expected Result:**
    *   The tool blocks until messages arrive or timeout.
    *   All received messages are formatted and returned to the AI.

## Execution Summary
Once these scenarios are verified, the NATS MCP server is considered "Production Ready" for local developer integration.
