import Link from "next/link";

export default function DashboardPage() {
  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <p>Testing websocket persistence.</p>

      <Link
        href="/"
        className="text-blue-600 underline"
      >
        Back to Home
      </Link>
    </div>
  );
}