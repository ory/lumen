type Params = { params: { slug: string } };

export default async function ArticlePage({ params }: Params) {
  const article = await loadArticle(params.slug);
  return <main><h1>{article.title}</h1><div>{article.body}</div></main>;
}
