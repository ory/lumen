type Props = { searchParams: { to?: string } };

export default function ShortLinkRedirect({ searchParams }: Props) {
  const target = searchParams.to ?? '/';
  return <meta httpEquiv="refresh" content={`0;url=${target}`} />;
}
