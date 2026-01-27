export default {
  selector: {
    label: 'Add to Collection',
    none: 'No Collection',
  },
  create: {
    title: 'New Collection',
    name: 'Collection Name',
    namePlaceholder: 'Enter collection name',
    description: 'Description',
    descPlaceholder: 'Briefly describe this collection (optional)',
    nameRequired: 'Please enter collection name',
    success: 'Collection created',
    failed: 'Failed to create collection',
  },
  detail: {
    loadFailed: 'Failed to load collection',
    items: 'items',
    posts: 'Posts',
    works: 'Works',
    empty: 'No content in this collection',
  },
  delete: {
    title: 'Delete Collection',
    confirm: 'Are you sure you want to delete this collection? This cannot be undone.',
    success: 'Collection deleted',
    failed: 'Failed to delete collection',
  },
  nav: {
    prev: 'Previous',
    next: 'Next',
  },
  banner: {
    belongsTo: 'Part of Collection',
    totalCount: '{count} items',
    edit: 'Edit Collection',
    addToCollection: 'Add to Collection',
    updateSuccess: 'Collection updated',
    updateFailed: 'Failed to update collection',
  },
  favorite: {
    favorite: 'Favorite',
    favorited: 'Favorited',
    added: 'Added to favorites',
    removed: 'Removed from favorites',
    failed: 'Operation failed',
  },
}
