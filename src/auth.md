Build a complete NestJS JWT authentication system with the following modules and behavior:

## Project setup

- Install dependencies: @nestjs/passport, @nestjs/jwt, passport, passport-local, passport-jwt, bcrypt, @types/passport-local, @types/passport-jwt, @types/bcrypt

---

## Module structure

Create the following modules:

- AuthModule
- UsersModule

---

## Users

Create a User entity with:

- id (UUID, primary key)
- email (unique, string)
- passwordHash (string)
- refreshTokenHash (string, nullable)
- createdAt (timestamp)

Create UsersService with:

- create(email, passwordHash): saves a new user
- findByEmail(email): finds user by email
- updateRefreshToken(userId, hash): stores hashed refresh token (or null on logout)

---

## Auth flow

### Register — POST /auth/register

- Accept { email, password }
- Throw ConflictException if email already exists
- Hash password with bcrypt (saltRounds: 10)
- Call UsersService.create()
- Return { message: 'User registered successfully' }

### Login — POST /auth/login

- Protect with LocalAuthGuard (Passport LocalStrategy)
- LocalStrategy.validate(): find user by email, compare password with bcrypt, throw UnauthorizedException on failure
- On success: call AuthService.login(user)
  - Sign access token (JWT): payload { sub: user.id, email }, expiresIn: '15m'
  - Sign refresh token (JWT): payload { sub: user.id }, expiresIn: '7d'
  - Hash refresh token with bcrypt and store via UsersService.updateRefreshToken()
  - Return { accessToken, refreshToken }

### Protected route — GET /auth/profile

- Protect with JwtAuthGuard (Passport JwtStrategy)
- JwtStrategy: extract Bearer token from Authorization header, verify with JWT secret, return decoded payload
- Return req.user (the decoded payload)

### Refresh — POST /auth/refresh

- Accept { refreshToken } in request body
- Find user by sub from decoded refresh token (use JwtService.decode, not verify, to get sub first)
- Verify the refresh token against the stored hash using bcrypt.compare()
- If invalid or hash is null, throw ForbiddenException
- Rotate tokens: issue new accessToken + refreshToken, update stored hash
- Return { accessToken, refreshToken }

### Logout — POST /auth/logout

- Protect with JwtAuthGuard
- Call UsersService.updateRefreshToken(userId, null) to invalidate the refresh token
- Return { message: 'Logged out successfully' }

---

## Guards and strategies

- LocalStrategy extends PassportStrategy(Strategy from passport-local)
- JwtStrategy extends PassportStrategy(Strategy from passport-jwt), using ExtractJwt.fromAuthHeaderAsBearerToken()
- LocalAuthGuard extends AuthGuard('local')
- JwtAuthGuard extends AuthGuard('jwt')
- Create a @CurrentUser() param decorator that extracts req.user from the ExecutionContext

---

## Configuration

- Store JWT_SECRET and JWT_REFRESH_SECRET in a .env file
- Use @nestjs/config (ConfigModule.forRoot()) to load env vars
- Pass secrets via ConfigService into JwtModule and the strategies

---

## Error handling

- ConflictException for duplicate email on register
- UnauthorizedException for invalid credentials on login
- ForbiddenException for invalid/expired refresh token
- Let JwtAuthGuard handle 401 for protected routes automatically

---

## Testing

After building, verify the full flow works with these curl commands:

1. Register:
   curl -X POST http://localhost:3000/auth/register -H "Content-Type: application/json" -d '{"email":"test@example.com","password":"secret123"}'

2. Login:
   curl -X POST http://localhost:3000/auth/login -H "Content-Type: application/json" -d '{"email":"test@example.com","password":"secret123"}'

3. Access protected route (replace TOKEN):
   curl http://localhost:3000/auth/profile -H "Authorization: Bearer TOKEN"

4. Refresh tokens (replace REFRESH_TOKEN):
   curl -X POST http://localhost:3000/auth/refresh -H "Content-Type: application/json" -d '{"refreshToken":"REFRESH_TOKEN"}'

5. Logout (replace TOKEN):
   curl -X POST http://localhost:3000/auth/logout -H "Authorization: Bearer TOKEN"

---

## Final checklist

- All modules properly imported and providers registered
- AppModule imports ConfigModule, MongooseModule, AuthModule, UsersModule
- .env file created with JWT_SECRET and JWT_REFRESH_SECRET values
- App runs on port 3000 with `npm run start:dev`
