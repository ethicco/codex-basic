import { NextRequest } from "next/server";
import { handleLogout } from "@/lib/auth-proxy";

export async function POST(request: NextRequest) {
  return handleLogout(request);
}
