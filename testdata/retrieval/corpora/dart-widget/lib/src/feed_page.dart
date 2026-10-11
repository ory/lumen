import 'dart:async';

import 'package:flutter/material.dart';

import 'post.dart';
import 'post_repository.dart';

class FeedPage extends StatefulWidget {
  const FeedPage({super.key, required this.repository});

  final PostRepository repository;

  @override
  State<FeedPage> createState() => _FeedPageState();
}

class _FeedPageState extends State<FeedPage> {
  StreamSubscription<List<Post>>? _subscription;
  List<Post> _posts = const [];

  @override
  void initState() {
    super.initState();
    _subscription = widget.repository.watchPosts().listen(_onPosts);
  }

  void _onPosts(List<Post> posts) {
    // The stream can deliver after the page was popped.
    if (!mounted) return;
    setState(() {
      _posts = posts;
    });
  }

  @override
  void dispose() {
    _subscription?.cancel();
    _subscription = null;
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: _posts.length,
      itemBuilder: (context, index) => ListTile(
        title: Text(_posts[index].author),
        subtitle: Text(_posts[index].body),
      ),
    );
  }
}
