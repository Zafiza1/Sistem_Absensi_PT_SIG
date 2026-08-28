import { redirect } from "next/navigation";

// The root route has no content of its own — /dashboard's layout already
// handles bouncing an unauthenticated visitor to /login, so this only
// needs to pick a destination, not duplicate that check.
export default function RootPage() {
  redirect("/dashboard");
}
