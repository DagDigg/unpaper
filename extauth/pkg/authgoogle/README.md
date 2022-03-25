# Google login flow

1.  FE: User visit /login
2.  FE: User clicks 'Login with Google'
3.  BE: Code verifier and code challenge are created
4.  BE: Auth endpoint is created with code challenge and returned back to FE alongside an httpOnly Secure SameSite `Code-Verifier` session cookie
5.  FE: visits auth endpoint, user inserts credentials on consent screen
6.  FE: `/callback` with `code` and `code verifier`
7.  BE: Exchange `code` and `code verifier` with access-token
8.  BE: access-token is ignored, since we don't need to interact with google APIs
9.  BE: a new session is created, stored in-memory, and sent as Secure httpOnly SameSite cookie

From now, the session will be carried over by the frontend, and checked on backend.

The in-memory store will resemble something like this:

|                 ByUserID                 |                    BySessionID                     |
| :--------------------------------------: | :------------------------------------------------: |
| 'bob-user-id': ['bob-session-id-1', ...] | 'bob-session-id-1': {userID: 'bob-user-id-1', ...} |

In this way, If I want to perform a 'logout from all devices', I iterate over `ByUserID`, and remove each session
