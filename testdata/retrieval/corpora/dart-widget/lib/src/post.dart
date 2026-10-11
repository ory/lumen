/// A single timeline entry.
class Post {
  final String id;
  final String author;
  final String body;
  final DateTime createdAt;

  const Post({
    required this.id,
    required this.author,
    required this.body,
    required this.createdAt,
  });
}
