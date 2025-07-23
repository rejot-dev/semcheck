function handleGetUser(r: Request): Response {
  const user = getUser(r.userId);

  if (user) {
    return new Response(user, { status: 201 }); // [!code error]
  }
  return new Response(null, { status: 404 });
}
