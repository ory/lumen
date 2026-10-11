import 'dart:async';

import 'post.dart';

/// Keeps the in-memory timeline and broadcasts every change.
class PostRepository {
  final _controller = StreamController<List<Post>>.broadcast();
  final List<Post> _posts = [];

  /// Emits the full list of posts whenever it changes.
  Stream<List<Post>> watchPosts() => _controller.stream;

  void publish(Post post) {
    _posts.insert(0, post);
    _controller.add(List.unmodifiable(_posts));
  }

  /// Closes the broadcast controller. Listeners receive a done event.
  Future<void> close() => _controller.close();
}
