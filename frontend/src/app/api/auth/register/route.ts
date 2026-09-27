import { NextRequest } from "next/server";
import { handleCredentials } from "@/lib/auth-proxy";

export async function POST(request: NextRequest) {
  return handleCredentials(request, "/api/auth/register");
}
