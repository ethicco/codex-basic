import { NextRequest } from "next/server";

import { handleSession } from "@/lib/auth-proxy";

export async function GET(request: NextRequest) {
  return handleSession(request);
}
