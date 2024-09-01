1. run the test case in `crosstalk_test.go`.
1. note that it passes.
1. comment out the assignment of `pool.Transport` in the `groupcache.HTTPPoolOptions`.
1. note that it fails.

this shows that when groupcache instances with out-of-sync peer lists may propagate requests to each other and wait on each other to fulfill the requests. all requests for an object "stuck" in this way will hang until the original context times out.

instead, the library should behave such that when handling a request on the groupcache handler, no further peers should be consulted to prevent a lockup. in this example, the roundtripper returns an error which `groupcache` handles gracefully:

- the request comes in and `group.Get()` is called ([code](https://github.com/mailgun/groupcache/blob/9f417fbc4f99eb58e51f8be01b1ac627a83a348f/http.go#L227))
- `Get()` attempts to find the object locally. if not found, it'll load it from the group ([code](https://github.com/mailgun/groupcache/blob/9f417fbc4f99eb58e51f8be01b1ac627a83a348f/groupcache.go#L257))
- `load()` attempts to get the object from a peer ([code](https://github.com/mailgun/groupcache/blob/9f417fbc4f99eb58e51f8be01b1ac627a83a348f/groupcache.go#L382))
- the crosstalk transport defined in this project returns a new error in the roundtripper; no actual http request is made to the peer
- the error [is none of these](https://github.com/mailgun/groupcache/blob/9f417fbc4f99eb58e51f8be01b1ac627a83a348f/groupcache.go#L397-L423) so the execution continues and the node [just gets the value itself](https://github.com/mailgun/groupcache/blob/9f417fbc4f99eb58e51f8be01b1ac627a83a348f/groupcache.go#L426).

a similar fix can be applied to other forks of `groupcache`, as well as the original.