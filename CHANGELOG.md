# v0.0.2
* Add a mutex lock around the initial BoostrapHealthCheck. While this forces all requests for the initial health check to run sequentially, it ensures we can't inadvertently revert state after a successful transition to NodeHealthCheck.
* Minor refactor around environment variable lookups.

# v0.0.1
* Initial release