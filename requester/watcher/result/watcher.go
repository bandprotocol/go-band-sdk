package result

// Get request id from channel poll until resolved
// Send success request to result channel
// Send fail/timeout request to retry handler (via channel)
